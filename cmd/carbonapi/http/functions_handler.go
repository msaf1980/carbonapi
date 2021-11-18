package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-graphite/carbonapi/carbonapipb"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	utilctx "github.com/go-graphite/carbonapi/util/ctx"
	"github.com/lomik/zapwriter"
	"go.uber.org/zap"
)

func functionsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement helper for specific functions
	t0 := time.Now()
	username, _, _ := r.BasicAuth()

	srcIP, srcPort := splitRemoteAddr(r.RemoteAddr)

	accessLogger := zapwriter.Logger("access")
	var accessLogDetails = carbonapipb.AccessLogDetails{
		Handler:        "functions",
		Username:       username,
		URL:            r.URL.RequestURI(),
		PeerIP:         srcIP,
		PeerPort:       srcPort,
		Host:           r.Host,
		Referer:        r.Referer(),
		URI:            r.RequestURI,
		RequestHeaders: utilctx.GetLogHeaders(r.Context()),
	}

	logAsError := false
	defer func() {
		deferredAccessLogging(accessLogger, &accessLogDetails, t0, logAsError)
	}()

	ApiMetrics.Requests.Add(1)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest)+": "+err.Error(), http.StatusBadRequest)
		accessLogDetails.HTTPCode = http.StatusBadRequest
		accessLogDetails.Reason = err.Error()
		logAsError = true
		return
	}

	grouped := false
	nativeOnly := false
	groupedStr := r.FormValue("grouped")
	prettyStr := r.FormValue("pretty")
	nativeOnlyStr := r.FormValue("nativeOnly")
	var marshaler func(interface{}) ([]byte, error)

	if groupedStr == "1" {
		grouped = true
	}

	if prettyStr == "1" {
		marshaler = func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "\t")
		}
	} else {
		marshaler = json.Marshal
	}

	if nativeOnlyStr == "1" {
		nativeOnly = true
	}

	path := strings.Split(r.URL.EscapedPath(), "/")
	function := ""
	if len(path) >= 3 {
		function = path[2]
	}

	var b []byte
	if !nativeOnly {
		parser.FunctionMD.RLock()
		if function != "" {
			b, err = marshaler(parser.FunctionMD.Descriptions[function])
		} else if grouped {
			b, err = marshaler(parser.FunctionMD.DescriptionsGrouped)
		} else {
			b, err = marshaler(parser.FunctionMD.Descriptions)
		}
		parser.FunctionMD.RUnlock()
	} else {
		parser.FunctionMD.RLock()
		if function != "" {
			if !parser.FunctionMD.Descriptions[function].Proxied {
				b, err = marshaler(parser.FunctionMD.Descriptions[function])
			} else {
				err = fmt.Errorf("%v is proxied to graphite-web and nativeOnly was specified", function)
			}
		} else if grouped {
			descGrouped := make(map[string]map[string]types.FunctionDescription)
			for groupName, description := range parser.FunctionMD.DescriptionsGrouped {
				desc := make(map[string]types.FunctionDescription)
				for f, d := range description {
					if d.Proxied {
						continue
					}
					desc[f] = d
				}
				if len(desc) > 0 {
					descGrouped[groupName] = desc
				}
			}
			b, err = marshaler(descGrouped)
		} else {
			desc := make(map[string]types.FunctionDescription)
			for f, d := range parser.FunctionMD.Descriptions {
				if d.Proxied {
					continue
				}
				desc[f] = d
			}
			b, err = marshaler(desc)
		}
		parser.FunctionMD.RUnlock()
	}

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		accessLogDetails.HTTPCode = http.StatusInternalServerError
		accessLogDetails.Reason = err.Error()
		logAsError = true
		return
	}

	_, _ = w.Write(b)
	accessLogDetails.Runtime = time.Since(t0).Seconds()
	accessLogDetails.HTTPCode = http.StatusOK

	accessLogger.Info("request served", zap.Any("data", accessLogDetails))
}
