package http

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ansel1/merry"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
	"github.com/lomik/zapwriter"
	"github.com/msaf1980/go-stringutils"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/go-graphite/carbonapi/carbonapipb"
	"github.com/go-graphite/carbonapi/cmd/carbonapi/config"
	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr"
	"github.com/go-graphite/carbonapi/expr/functions/cairo/png"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	utilctx "github.com/go-graphite/carbonapi/util/ctx"
	"github.com/go-graphite/carbonapi/zipper/helper"
)

func cleanupParams(r *http.Request) {
	// make sure the cache key doesn't say noCache, because it will never hit
	r.Form.Del("noCache")

	// jsonp callback names are frequently autogenerated and hurt our cache
	r.Form.Del("jsonp")

	// Strip some cache-busters.  If you don't want to cache, use noCache=1
	r.Form.Del("_salt")
	r.Form.Del("_ts")
	r.Form.Del("_t") // Used by jquery.graphite.js
}

func getCacheTimeout(logger *zap.Logger, r *http.Request, now32, until32 int64, duration time.Duration, cacheConfig *config.CacheConfig) int32 {
	if tstr := r.FormValue("cacheTimeout"); tstr != "" {
		t, err := strconv.Atoi(tstr)
		if err != nil {
			logger.Error("failed to parse cacheTimeout",
				zap.String("cache_string", tstr),
				zap.Error(err),
			)
		} else {
			return int32(t)
		}
	}
	if now32 == 0 || cacheConfig.ShortTimeoutSec == 0 || cacheConfig.ShortDuration == 0 {
		return cacheConfig.DefaultTimeoutSec
	}
	if duration > cacheConfig.ShortDuration || now32-until32 > cacheConfig.ShortUntilOffsetSec {
		return cacheConfig.DefaultTimeoutSec
	}
	// short cache ttl
	return cacheConfig.ShortTimeoutSec
}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	uid := uuid.NewV4()

	// TODO: Migrate to context.WithTimeout
	// ctx, _ := context.WithTimeout(context.TODO(), config.Config.ZipperTimeout)
	ctx := utilctx.SetUUID(r.Context(), uid.String())
	username, _, _ := r.BasicAuth()
	requestHeaders := utilctx.GetLogHeaders(ctx)

	logger := zapwriter.Logger("render").With(
		zap.String("carbonapi_uuid", uid.String()),
		zap.String("username", username),
		zap.Any("request_headers", requestHeaders),
	)

	srcIP, srcPort := splitRemoteAddr(r.RemoteAddr)

	accessLogger := zapwriter.Logger("access")
	var accessLogDetails = &carbonapipb.AccessLogDetails{
		Handler:        "render",
		Username:       username,
		CarbonapiUUID:  uid.String(),
		URL:            r.URL.RequestURI(),
		PeerIP:         srcIP,
		PeerPort:       srcPort,
		Host:           r.Host,
		Referer:        r.Referer(),
		URI:            r.RequestURI,
		RequestHeaders: requestHeaders,
	}

	logAsError := false
	defer func() {
		deferredAccessLogging(accessLogger, accessLogDetails, t0, logAsError)
	}()

	err := r.ParseForm()
	if err != nil {
		setError(w, accessLogDetails, err.Error(), http.StatusBadRequest, uid.String())
		logAsError = true
		return
	}

	targets := r.Form["target"]
	from := r.FormValue("from")
	until := r.FormValue("until")
	template := r.FormValue("template")
	maxDataPoints, _ := strconv.ParseInt(r.FormValue("maxDataPoints"), 10, 64)
	ctx = utilctx.SetMaxDatapoints(ctx, maxDataPoints)
	useCache := !parser.TruthyBool(r.FormValue("noCache"))
	noNullPoints := parser.TruthyBool(r.FormValue("noNullPoints"))
	// status will be checked later after we'll setup everything else
	format, ok, formatRaw := getFormat(r, pngFormat)

	var jsonp string

	if format == jsonFormat {
		// TODO(dgryski): check jsonp only has valid characters
		jsonp = r.FormValue("jsonp")
	}

	timestampFormat := strings.ToLower(r.FormValue("timestampFormat"))
	if timestampFormat == "" {
		timestampFormat = "s"
	}

	timestampMultiplier := int64(1)
	switch timestampFormat {
	case "s":
		timestampMultiplier = 1
	case "ms", "millisecond", "milliseconds":
		timestampMultiplier = 1000
	case "us", "microsecond", "microseconds":
		timestampMultiplier = 1000000
	case "ns", "nanosecond", "nanoseconds":
		timestampMultiplier = 1000000000
	default:
		setError(w, accessLogDetails, "unsupported timestamp format, supported: 's', 'ms', 'us', 'ns'", http.StatusBadRequest, uid.String())
		logAsError = true
		return
	}

	now := timeNow()
	now32 := now.Unix()

	cleanupParams(r)

	// normalize from and until values
	qtz := r.FormValue("tz")
	from32 := date.DateParamToEpoch(from, qtz, now.Add(-24*time.Hour).Unix(), config.Config.DefaultTimeZone)
	until32 := date.DateParamToEpoch(until, qtz, now.Unix(), config.Config.DefaultTimeZone)

	var (
		responseCacheKey     string
		responseCacheTimeout int32
		backendCacheTimeout  int32
	)

	duration := time.Second * time.Duration(until32-from32)
	if len(config.Config.TruncateTime) > 0 {
		from32 = timestampTruncate(from32, duration, config.Config.TruncateTime)
		until32 = timestampTruncate(until32, duration, config.Config.TruncateTime)
		// recalc duration
		duration = time.Second * time.Duration(until32-from32)
		responseCacheKey = responseCacheComputeKey(from32, until32, targets, formatRaw, maxDataPoints, noNullPoints, template)
		if useCache {
			responseCacheTimeout = getCacheTimeout(logger, r, now32, until32, duration, &config.Config.ResponseCacheConfig)
			backendCacheTimeout = getCacheTimeout(logger, r, now32, until32, duration, &config.Config.BackendCacheConfig)
		}
	} else {
		responseCacheKey = r.Form.Encode()
		if useCache {
			responseCacheTimeout = getCacheTimeout(logger, r, now32, until32, duration, &config.Config.ResponseCacheConfig)
			backendCacheTimeout = getCacheTimeout(logger, r, now32, until32, duration, &config.Config.BackendCacheConfig)
		}
	}

	accessLogDetails.UseCache = useCache
	accessLogDetails.FromRaw = from
	accessLogDetails.From = from32
	accessLogDetails.UntilRaw = until
	accessLogDetails.Until = until32
	accessLogDetails.Tz = qtz
	accessLogDetails.CacheTimeout = responseCacheTimeout
	accessLogDetails.Format = formatRaw
	accessLogDetails.Targets = targets

	if !ok || !format.ValidRenderFormat() {
		setError(w, accessLogDetails, "unsupported format specified: "+formatRaw, http.StatusBadRequest, uid.String())
		logAsError = true
		return
	}

	if format == protoV3Format {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			setError(w, accessLogDetails, "failed to parse message body: "+err.Error(), http.StatusBadRequest, uid.String())
			return
		}

		var pv3Request pb.MultiFetchRequest
		err = pv3Request.Unmarshal(body)

		if err != nil {
			setError(w, accessLogDetails, "failed to parse message body: "+err.Error(), http.StatusBadRequest, uid.String())
			return
		}

		from32 = pv3Request.Metrics[0].StartTime
		until32 = pv3Request.Metrics[0].StopTime
		targets = make([]string, len(pv3Request.Metrics))
		for i, r := range pv3Request.Metrics {
			targets[i] = r.PathExpression
		}
	}

	if queryLengthLimitExceeded(targets, config.Config.MaxQueryLength) {
		setError(w, accessLogDetails, "total target length limit exceeded", http.StatusBadRequest, uid.String())
		logAsError = true
		return
	}

	if useCache {
		tc := time.Now()
		response, err := config.Config.ResponseCache.Get(responseCacheKey)
		td := time.Since(tc).Nanoseconds()
		ApiMetrics.RequestsCacheOverheadNS.Add(uint64(td))

		accessLogDetails.CarbonzipperResponseSizeBytes = 0
		accessLogDetails.CarbonapiResponseSizeBytes = int64(len(response))

		if err == nil {
			ApiMetrics.RequestCacheHits.Add(1)
			w.Header().Set("X-Carbonapi-Request-Cached", strconv.FormatInt(int64(responseCacheTimeout), 10))
			writeResponse(w, http.StatusOK, response, format, jsonp, uid.String())
			accessLogDetails.FromCache = true
			return
		}
		ApiMetrics.RequestCacheMisses.Add(1)
	}

	if from32 >= until32 {
		setError(w, accessLogDetails, "Invalid or empty time range", http.StatusBadRequest, uid.String())
		logAsError = true
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic during eval:",
				zap.String("cache_key", responseCacheKey),
				zap.Any("reason", r),
				zap.Stack("stack"),
			)
			logAsError = true
			var answer string
			if config.Config.HTTPResponseStackTrace {
				answer = fmt.Sprintf("%v\nStack trace: %v", r, zap.Stack("").String)
			} else {
				answer = fmt.Sprint(r)
			}
			setError(w, accessLogDetails, answer, http.StatusInternalServerError, uid.String())
		}
	}()

	errors := make(map[string]merry.Error)

	var backendCacheKey string
	if len(config.Config.TruncateTime) > 0 {
		backendCacheKey = backendCacheComputeKeyAbs(from32, until32, targets, maxDataPoints, noNullPoints)
	} else {
		backendCacheKey = backendCacheComputeKey(from, until, targets, maxDataPoints, noNullPoints)
	}

	results, err := backendCacheFetchResults(logger, useCache, backendCacheKey, accessLogDetails)

	if err != nil {
		ApiMetrics.BackendCacheMisses.Add(1)

		results = make([]*types.MetricData, 0)
		values := make(map[parser.MetricRequest][]*types.MetricData)

		if config.Config.CombineMultipleTargetsInOne && len(targets) > 0 {
			exprs := make([]parser.Expr, 0, len(targets))
			for _, target := range targets {
				exp, e, err := parser.ParseExpr(target)
				if err != nil || e != "" {
					msg := buildParseErrorString(target, e, err)
					setError(w, accessLogDetails, msg, http.StatusBadRequest, uid.String())
					logAsError = true
					return
				}
				exprs = append(exprs, exp)
			}

			ApiMetrics.RenderRequests.Add(1)

			result, errs := expr.FetchAndEvalExprs(ctx, config.Config.Evaluator, exprs, from32, until32, values)
			if errs != nil {
				errors = errs
			}

			results = append(results, result...)
		} else {
			for _, target := range targets {
				exp, e, err := parser.ParseExpr(target)
				if err != nil || e != "" {
					msg := buildParseErrorString(target, e, err)
					setError(w, accessLogDetails, msg, http.StatusBadRequest, uid.String())
					logAsError = true
					return
				}

				ApiMetrics.RenderRequests.Add(1)

				result, err := expr.FetchAndEvalExp(ctx, config.Config.Evaluator, exp, from32, until32, values)
				if err != nil {
					errors[target] = merry.Wrap(err)
					// if config.Config.Upstreams.RequireSuccessAll {
					// 	break
					// }
				}

				results = append(results, result...)
			}
		}

		if len(errors) == 0 && backendCacheTimeout > 0 {
			w.Header().Set("X-Carbonapi-Backend-Cached", strconv.FormatInt(int64(backendCacheTimeout), 10))
			backendCacheStoreResults(logger, backendCacheKey, results, backendCacheTimeout)
		}
	}

	size := 0
	for _, result := range results {
		size += result.Size()
	}

	var body []byte

	returnCode := http.StatusOK
	if len(results) == 0 || (len(errors) > 0 && config.Config.Upstreams.RequireSuccessAll) {
		// Obtain error code from the errors
		// In case we have only "Not Found" errors, result should be 404
		// Otherwise it should be 500
		var errMsgs []string
		returnCode, errMsgs = helper.MergeHttpErrorMap(errors)
		logger.Debug("error response or no response", zap.Strings("error", errMsgs))
		// Allow override status code for 404-not-found replies.
		if returnCode == http.StatusNotFound {
			returnCode = config.Config.NotFoundStatusCode
		}

		if returnCode == http.StatusBadRequest || returnCode == http.StatusNotFound || returnCode == http.StatusForbidden || returnCode >= 500 {
			setError(w, accessLogDetails, strings.Join(errMsgs, ","), returnCode, uid.String())
			logAsError = true
			return
		}
	}

	switch format {
	case jsonFormat:
		if maxDataPoints != 0 {
			types.ConsolidateJSON(maxDataPoints, results)
			accessLogDetails.MaxDataPoints = maxDataPoints
		}

		body = types.MarshalJSON(results, timestampMultiplier, noNullPoints)
	case protoV2Format:
		body, err = types.MarshalProtobufV2(results)
		if err != nil {
			setError(w, accessLogDetails, err.Error(), http.StatusInternalServerError, uid.String())
			logAsError = true
			return
		}
	case protoV3Format:
		body, err = types.MarshalProtobufV3(results)
		if err != nil {
			setError(w, accessLogDetails, err.Error(), http.StatusInternalServerError, uid.String())
			logAsError = true
			return
		}
	case rawFormat:
		body = types.MarshalRaw(results)
	case csvFormat:
		body = types.MarshalCSV(results)
	case pickleFormat:
		body = types.MarshalPickle(results)
	case pngFormat:
		body = png.MarshalPNGRequest(r, results, template)
	case svgFormat:
		body = png.MarshalSVGRequest(r, results, template)
	}

	accessLogDetails.Metrics = targets
	accessLogDetails.CarbonzipperResponseSizeBytes = int64(size)
	accessLogDetails.CarbonapiResponseSizeBytes = int64(len(body))

	writeResponse(w, returnCode, body, format, jsonp, uid.String())

	if len(results) != 0 {
		tc := time.Now()
		config.Config.ResponseCache.Set(responseCacheKey, body, responseCacheTimeout)
		td := time.Since(tc).Nanoseconds()
		ApiMetrics.RequestsCacheOverheadNS.Add(uint64(td))
	}

	gotErrors := len(errors) > 0
	accessLogDetails.HaveNonFatalErrors = gotErrors
}

func responseCacheComputeKey(from, until int64, targets []string, format string, maxDataPoints int64, noNullPoints bool, template string) string {
	var responseCacheKey stringutils.Builder
	responseCacheKey.Grow(256)
	responseCacheKey.WriteString("from:")
	responseCacheKey.WriteInt(from, 10)
	responseCacheKey.WriteString(" until:")
	responseCacheKey.WriteInt(until, 10)
	responseCacheKey.WriteString(" targets:")
	responseCacheKey.WriteString(strings.Join(targets, ","))
	responseCacheKey.WriteString(" format:")
	responseCacheKey.WriteString(format)
	if maxDataPoints > 0 {
		responseCacheKey.WriteString(" maxDataPoints:")
		responseCacheKey.WriteInt(maxDataPoints, 10)
	}
	if noNullPoints {
		responseCacheKey.WriteString(" noNullPoints")
	}
	if len(template) > 0 {
		responseCacheKey.WriteString(" template:")
		responseCacheKey.WriteString(template)
	}
	return responseCacheKey.String()
}

func backendCacheComputeKey(from, until string, targets []string, maxDataPoints int64, noNullPoints bool) string {
	var backendCacheKey stringutils.Builder
	backendCacheKey.WriteString("from:")
	backendCacheKey.WriteString(from)
	backendCacheKey.WriteString(" until:")
	backendCacheKey.WriteString(until)
	backendCacheKey.WriteString(" targets:")
	backendCacheKey.WriteString(strings.Join(targets, ","))
	if maxDataPoints > 0 {
		backendCacheKey.WriteString(" maxDataPoints:")
		backendCacheKey.WriteInt(maxDataPoints, 10)
	}
	if noNullPoints {
		backendCacheKey.WriteString(" noNullPoints")
	}
	return backendCacheKey.String()
}

func backendCacheComputeKeyAbs(from, until int64, targets []string, maxDataPoints int64, noNullPoints bool) string {
	var backendCacheKey stringutils.Builder
	backendCacheKey.Grow(128)
	backendCacheKey.WriteString("from:")
	backendCacheKey.WriteInt(from, 10)
	backendCacheKey.WriteString(" until:")
	backendCacheKey.WriteInt(until, 10)
	backendCacheKey.WriteString(" targets:")
	backendCacheKey.WriteString(strings.Join(targets, ","))
	if maxDataPoints > 0 {
		backendCacheKey.WriteString(" maxDataPoints:")
		backendCacheKey.WriteInt(maxDataPoints, 10)
	}
	if noNullPoints {
		backendCacheKey.WriteString(" noNullPoints")
	}
	return backendCacheKey.String()
}

func backendCacheFetchResults(logger *zap.Logger, useCache bool, backendCacheKey string, accessLogDetails *carbonapipb.AccessLogDetails) ([]*types.MetricData, error) {
	if !useCache {
		return nil, errors.New("useCache is false")
	}

	backendCacheResults, err := config.Config.BackendCache.Get(backendCacheKey)

	if err != nil {
		return nil, err
	}

	var results []*types.MetricData
	cacheDecodingBuf := bytes.NewBuffer(backendCacheResults)
	dec := gob.NewDecoder(cacheDecodingBuf)
	err = dec.Decode(&results)

	if err != nil {
		logger.Error("Error decoding cached backend results")
		return nil, err
	}

	accessLogDetails.UsedBackendCache = true
	ApiMetrics.BackendCacheHits.Add(uint64(1))

	return results, nil
}

func backendCacheStoreResults(logger *zap.Logger, backendCacheKey string, results []*types.MetricData, backendCacheTimeout int32) {
	var serializedResults bytes.Buffer
	enc := gob.NewEncoder(&serializedResults)
	err := enc.Encode(results)

	if err != nil {
		logger.Error("Error encoding backend results for caching")
		return
	}

	config.Config.BackendCache.Set(backendCacheKey, serializedResults.Bytes(), backendCacheTimeout)
}
