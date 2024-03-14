package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ansel1/merry"
	uuid "github.com/satori/go.uuid"

	"github.com/go-graphite/carbonapi/carbonapipb"
	"github.com/go-graphite/carbonapi/cmd/carbonapi/config"
	utilctx "github.com/go-graphite/carbonapi/util/ctx"
	"github.com/go-graphite/carbonapi/zipper/types"
	"github.com/lomik/zapwriter"
	"go.uber.org/zap"
)

func tagHandler(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	uuid := uuid.NewV4()
	carbonapiUUID := uuid.String()

	// TODO: Migrate to context.WithTimeout
	ctx := utilctx.SetUUID(r.Context(), carbonapiUUID)
	requestHeaders := utilctx.GetLogHeaders(ctx)
	username, _, _ := r.BasicAuth()

	logger := zapwriter.Logger("tag").With(
		zap.String("carbonapi_uuid", carbonapiUUID),
		zap.String("username", username),
		zap.Any("request_headers", requestHeaders),
	)

	srcIP, srcPort := splitRemoteAddr(r.RemoteAddr)

	accessLogger := zapwriter.Logger("access")
	var accessLogDetails = &carbonapipb.AccessLogDetails{
		Handler:        "tags",
		Username:       username,
		CarbonapiUUID:  carbonapiUUID,
		URL:            r.URL.Path,
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
		logAsError = true
		setError(w, accessLogDetails, err.Error(), http.StatusBadRequest, carbonapiUUID)
		return
	}

	prettyStr := r.FormValue("pretty")
	limit := int64(-1)
	limitStr := r.FormValue("limit")
	if limitStr != "" {
		limit, err = strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			logger.Debug("error parsing limit, ignoring",
				zap.String("limit", r.FormValue("limit")),
				zap.Error(err),
			)
			limit = -1
		}
	}

	q := r.URL.Query()
	q.Del("pretty")
	rawQuery := q.Encode()

	if queryLengthLimitExceeded(r.Form["query"], config.Config.MaxQueryLength) {
		setError(w, accessLogDetails, "query length limit exceeded", http.StatusBadRequest, uuid.String())
		logAsError = true
		return
	}

	// TODO(civil): Implement caching
	var res []string
	if strings.HasSuffix(r.URL.Path, "tags") || strings.HasSuffix(r.URL.Path, "tags/") {
		res, err = config.Config.ZipperInstance.TagNames(ctx, rawQuery, limit)
	} else if strings.HasSuffix(r.URL.Path, "values") || strings.HasSuffix(r.URL.Path, "values/") {
		res, err = config.Config.ZipperInstance.TagValues(ctx, rawQuery, limit)
	} else {
		setError(w, accessLogDetails, http.StatusText(http.StatusNotFound), http.StatusNotFound, carbonapiUUID)
		return
	}

	// TODO(civil): Implement stats
	if err != nil && !merry.Is(err, types.ErrNoMetricsFetched) && !merry.Is(err, types.ErrNonFatalErrors) {
		code := merry.HTTPCode(err)
		logAsError = true
		setError(w, accessLogDetails, err.Error(), code, carbonapiUUID)
		return
	}

	var b []byte
	if prettyStr == "1" {
		b, err = json.MarshalIndent(res, "", "\t")
	} else {
		b, err = json.Marshal(res)
	}

	if err != nil {
		logAsError = true
		setError(w, accessLogDetails, err.Error(), http.StatusInternalServerError, carbonapiUUID)
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.Header().Set(ctxHeaderUUID, carbonapiUUID)
	_, _ = w.Write(b)
	accessLogDetails.Runtime = time.Since(t0).Seconds()
	accessLogDetails.HTTPCode = http.StatusOK
}
