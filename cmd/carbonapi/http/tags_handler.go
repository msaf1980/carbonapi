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

	// TODO: Migrate to context.WithTimeout
	ctx := r.Context()
	requestHeaders := utilctx.GetLogHeaders(ctx)
	username, _, _ := r.BasicAuth()

	logger := zapwriter.Logger("tag").With(
		zap.String("carbonapi_uuid", uuid.String()),
		zap.String("username", username),
		zap.Any("request_headers", requestHeaders),
	)

	srcIP, srcPort := splitRemoteAddr(r.RemoteAddr)

	accessLogger := zapwriter.Logger("access")
	var accessLogDetails = &carbonapipb.AccessLogDetails{
		Handler:        "tags",
		Username:       username,
		CarbonapiUUID:  uuid.String(),
		URL:            r.URL.Path,
		PeerIP:         srcIP,
		PeerPort:       srcPort,
		Host:           r.Host,
		Referer:        r.Referer(),
		URI:            r.RequestURI,
		RequestHeaders: requestHeaders,
	}

	ApiMetrics.Requests.Add(1)
	ApiMetrics.FindRequests.Add(1)

	logAsError := false
	defer func() {
		deferredAccessLogging(accessLogger, accessLogDetails, t0, logAsError)
		if config.Config.Graphite.ExtendedStat {
			if !accessLogDetails.FromCache {
				ApiMetrics.FindRequestsTime.Add(accessLogDetails.Runtime)
			}
			if accessLogDetails.CarbonapiResponseSizeBytes > 0 {
				ApiMetrics.FindRequestsSize.Add(float64(accessLogDetails.CarbonapiResponseSizeBytes))
			}
		}
	}()

	err := r.ParseForm()
	if err != nil {
		ApiMetrics.Errors.Add(1)
		ApiMetrics.FindErrors.Add(1)
		if config.Config.Graphite.ExtendedStat {
			ApiMetrics.FindCounter400.Add(1)
		}
		setError(w, accessLogDetails, "form parse error", "form parse error: "+err.Error(), http.StatusBadRequest)
		logAsError = true
		// w.Header().Set("Content-Type", contentTypeJSON)
		// _, _ = w.Write([]byte{'[', ']'})
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

	// TODO(civil): Implement caching
	var res []string
	if strings.HasSuffix(r.URL.Path, "tags") || strings.HasSuffix(r.URL.Path, "tags/") {
		res, err = config.Config.ZipperInstance.TagNames(ctx, rawQuery, limit)
	} else if strings.HasSuffix(r.URL.Path, "values") || strings.HasSuffix(r.URL.Path, "values/") {
		res, err = config.Config.ZipperInstance.TagValues(ctx, rawQuery, limit)
	} else {
		if config.Config.Graphite.ExtendedStat {
			ApiMetrics.FindCounter404.Add(1)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	returnCode := http.StatusOK

	// TODO(civil): Implement stats
	if err != nil {
		if merry.Is(err, types.ErrNoMetricsFetched) {
			returnCode = http.StatusNotFound
		} else {
			returnCode = merry.HTTPCode(err)
		}

		if config.Config.Graphite.ExtendedStat {
			switch returnCode {
			case http.StatusBadRequest:
				ApiMetrics.FindCounter400.Add(1)
			case http.StatusForbidden:
				ApiMetrics.FindCounter403.Add(1)
			case http.StatusNotFound:
				ApiMetrics.FindCounter404.Add(1)
			case http.StatusInternalServerError:
				ApiMetrics.FindCounter500.Add(1)
			case http.StatusBadGateway:
				ApiMetrics.FindCounter502.Add(1)
			case http.StatusServiceUnavailable:
				ApiMetrics.FindCounter503.Add(1)
			case http.StatusGatewayTimeout:
				ApiMetrics.FindCounter504.Add(1)
			}
		}

		if returnCode == http.StatusNotFound {
			returnCode = config.Config.NotFoundStatusCode
		}

		if returnCode == http.StatusForbidden {
			setError(w, accessLogDetails, "limits reached", err.Error(), returnCode)
		} else if returnCode != config.Config.NotFoundStatusCode {
			ApiMetrics.Errors.Add(1)
			ApiMetrics.FindErrors.Add(1)
			if returnCode == http.StatusInternalServerError {
				setError(w, accessLogDetails, "internal server error", err.Error(), returnCode)
			} else {
				setError(w, accessLogDetails, "failed to fetch data", err.Error(), returnCode)
			}
			if returnCode >= 500 {
				logAsError = true
			}
		}

		return
	}

	var b []byte
	if prettyStr == "1" {
		b, err = json.MarshalIndent(res, "", "\t")
	} else {
		b, err = json.Marshal(res)
	}

	if err != nil {
		ApiMetrics.Errors.Add(1)
		ApiMetrics.FindErrors.Add(1)
		if config.Config.Graphite.ExtendedStat {
			ApiMetrics.FindCounter500.Add(1)
		}
		setError(w, accessLogDetails, "internal error", "marhal response error: "+err.Error(), http.StatusInternalServerError)
		logAsError = true
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	_, err = w.Write(b)

	if config.Config.Graphite.ExtendedStat {
		ApiMetrics.FindCounter200.Add(1)
	}

	accessLogDetails.HTTPCode = http.StatusOK
}
