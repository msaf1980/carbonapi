package helper

import (
	"bytes"
	"context"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/ansel1/merry"
	"go.uber.org/zap"

	"github.com/go-graphite/carbonapi/limiter"
	"github.com/go-graphite/carbonapi/pkg/parser"
	util "github.com/go-graphite/carbonapi/util/ctx"
	"github.com/go-graphite/carbonapi/zipper/types"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	htmlTagStart = 60 // Unicode `<`
	htmlTagEnd   = 62 // Unicode `>`
)

// Aggressively strips HTML tags from a string.
// It will only keep anything between `>` and `<`.
func stripHtmlTags(s string, maxLen int) string {
	var n int
	if !strings.Contains(s, "<html>") {
		if maxLen == 0 || maxLen > len(s) {
			return s
		}
		return s[:maxLen]
	}
	// Setup a string builder and allocate enough memory for the new string.
	var builder strings.Builder
	if maxLen == 0 {
		n = len(s) + utf8.UTFMax
	} else {
		n = min(len(s), maxLen)
	}

	builder.Grow(n)

	in := false // True if we are inside an HTML tag.
	start := 0  // The index of the previous start tag character `<`
	end := 0    // The index of the previous end tag character `>`

	for i, c := range s {
		// If this is the last character and we are not in an HTML tag, save it.
		if (i+1) == len(s) && end >= start {
			builder.WriteString(s[end:])
		}

		if c == htmlTagStart {
			// Only update the start if we are not in a tag.
			// This make sure we strip out `<<br>` not just `<br>`
			if !in {
				start = i
			}
			in = true

			// Write the valid string between the close and start of the two tags.
			builder.WriteString(s[end:start])
			end = i + 1
		} else if c == htmlTagEnd {
			in = false
			end = i + 1
		}
	}
	s = strings.Trim(builder.String(), "\r\n")
	return s
}

func requestError(err error, server string) merry.Error {
	// with code InternalServerError by default, overwritten by custom error
	if merry.Is(err, context.DeadlineExceeded) {
		return types.ErrTimeoutExceeded.WithValue("server", server).WithCause(err)
	}
	if urlErr, ok := err.(*url.Error); ok {
		if netErr, ok := urlErr.Err.(*net.OpError); ok {
			return types.ErrBackendError.WithValue("server", server).WithCause(netErr)
		}
	}
	if netErr, ok := err.(*net.OpError); ok {
		return types.ErrBackendError.WithValue("server", server).WithCause(netErr)
	}
	return types.ErrResponceError.WithValue("server", server)
}

func MergeHttpErrorsCode(errors []merry.Error) (returnCode int) {
	returnCode = http.StatusNotFound
	for _, err := range errors {
		c := merry.RootCause(err)
		if c == nil {
			c = err
		}

		code := merry.HTTPCode(err)
		if code == http.StatusNotFound {
			continue
		} else if code == http.StatusInternalServerError && merry.Is(c, parser.ErrInvalidArg) {
			// check for invalid args, see applyByNode rewrite function
			code = http.StatusBadRequest
		}

		if code == http.StatusGatewayTimeout || code == http.StatusBadGateway {
			// simplify code, one error type for communications errors, all we can retry
			code = http.StatusServiceUnavailable
		}

		if code == http.StatusBadRequest {
			// The 400 is returned on wrong requests, e.g. non-existent functions
			returnCode = code
		} else if returnCode == http.StatusNotFound || code == http.StatusForbidden {
			// First error or access denied (may be limits or other)
			returnCode = code
		} else if code != http.StatusServiceUnavailable {
			returnCode = code
		}
	}

	return returnCode
}

func MergeHttpErrors(errors []merry.Error) (int, []string) {
	returnCode := http.StatusNotFound
	errMsgs := make([]string, 0)
	for _, err := range errors {
		c := merry.RootCause(err)
		if c == nil {
			c = err
		}

		code := merry.HTTPCode(err)
		if code == http.StatusNotFound {
			continue
		} else if code == http.StatusInternalServerError && merry.Is(c, parser.ErrInvalidArg) {
			// check for invalid args, see applyByNode rewrite function
			code = http.StatusBadRequest
		}

		if msg := merry.Message(c); len(msg) > 0 {
			errMsgs = append(errMsgs, strings.TrimRight(msg, "\n"))
		} else {
			errMsgs = append(errMsgs, c.Error())
		}

		if code == http.StatusGatewayTimeout || code == http.StatusBadGateway {
			// simplify code, one error type for communications errors, all we can retry
			code = http.StatusServiceUnavailable
		}

		if code == http.StatusBadRequest {
			// The 400 is returned on wrong requests, e.g. non-existent functions
			returnCode = code
		} else if returnCode == http.StatusNotFound || code == http.StatusForbidden {
			// First error or access denied (may be limits or other)
			returnCode = code
		} else if code != http.StatusServiceUnavailable {
			returnCode = code
		}
	}

	return returnCode, errMsgs
}

func MergeHttpErrorMap(errorsMap map[string]merry.Error) (int, []string) {
	errors := make([]merry.Error, len(errorsMap))
	i := 0
	for _, err := range errorsMap {
		errors[i] = err
		i++
	}

	return MergeHttpErrors(errors)
}

func HttpErrorByCode(err merry.Error) merry.Error {
	var returnErr merry.Error
	if err == nil {
		returnErr = types.ErrNoMetricsFetched
	} else {
		code := merry.HTTPCode(err)
		msg := stripHtmlTags(merry.Message(err), 0)
		if code == http.StatusForbidden {
			returnErr = types.ErrForbidden
			if len(msg) > 0 {
				// pass message to caller
				returnErr = returnErr.WithMessage(msg)
			}
		} else if code == http.StatusServiceUnavailable || code == http.StatusBadGateway || code == http.StatusGatewayTimeout {
			returnErr = types.ErrFailedToFetch.WithHTTPCode(code).WithMessage(msg)
		} else {
			returnErr = types.ErrFailed.WithHTTPCode(code).WithMessage(msg)
		}
	}

	return returnErr
}

type ServerResponse struct {
	Server   string
	Response []byte
}

type pickNext func(uint64, *zap.Logger) (string, uint64)

type HttpQuery struct {
	groupName  string
	servers    []string
	maxTries   int
	retryCodes map[int]bool
	lbMethod   types.LBMethod

	pickServer pickNext
	limiter    limiter.ServerLimiter
	client     *http.Client
	encoding   string

	counter uint64
}

func NewHttpQuery(groupName string, servers []string, maxTries int, retryCodes []int, lbMethod types.LBMethod, limiter limiter.ServerLimiter, client *http.Client, encoding string) *HttpQuery {
	if maxTries < 1 {
		maxTries = 1
	}
	c := &HttpQuery{
		groupName: groupName,
		servers:   servers,
		maxTries:  maxTries,
		limiter:   limiter,
		client:    client,
		encoding:  encoding,
	}
	if maxTries > 1 {
		if len(retryCodes) == 0 {
			retryCodes = []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}
		}
		c.retryCodes = make(map[int]bool)
		for _, code := range retryCodes {
			c.retryCodes[code] = true
		}
	}

	if len(c.servers) == 1 {
		c.pickServer = c.pickOneServer
	} else {
		c.pickServer = c.pickRandomServer
	}

	return c
}

func (c *HttpQuery) pickOneServer(n uint64, logger *zap.Logger) (string, uint64) {
	return c.servers[0], 0
}

func (c *HttpQuery) pickRandomServer(idx uint64, logger *zap.Logger) (string, uint64) {
	logger = logger.With(zap.String("function", "picker"))
	counter := atomic.AddUint64(&(c.counter), 1)
	// TODO: upstream failure detection
	if idx == math.MaxUint64 {
		idx = counter % uint64(len(c.servers))
	} else {
		idx = (idx + 1) % uint64(len(c.servers))
	}
	srv := c.servers[int(idx)]
	logger.Debug("picked",
		zap.Uint64("counter", counter),
		zap.Uint64("idx", idx),
		zap.String("server", srv),
	)

	return srv, idx
}

func (c *HttpQuery) doRequest(ctx context.Context, logger *zap.Logger, server, uri string, r types.Request) (*ServerResponse, merry.Error) {
	logger = logger.With(
		zap.String("function", "HttpQuery.doRequest"),
	)

	u, err := url.Parse(server + uri)
	if err != nil {
		return nil, merry.Here(err).WithValue("server", server)
	}

	var reader io.Reader
	var body []byte
	if r != nil {
		body, err = r.Marshal()
		if err != nil {
			return nil, merry.Here(err).WithValue("server", server)
		}
		if body != nil {
			reader = bytes.NewReader(body)
		}
	}
	logger = logger.With(
		zap.String("server", server),
		zap.String("name", c.groupName),
		zap.String("uri", u.String()),
	)

	// TODO: change to NewRequestWithContext
	req, err := http.NewRequest("GET", u.String(), reader)
	if err != nil {
		return nil, merry.Here(err).WithValue("server", server)
	}

	req.Header.Set("Accept", c.encoding)
	req = util.MarshalPassHeaders(ctx, util.MarshalCtx(ctx, util.MarshalCtx(ctx, req, util.HeaderUUIDZipper), util.HeaderUUIDAPI))

	logger.Debug("trying to get slot",
		zap.String("name", server),
	)
	err = c.limiter.Enter(ctx, server)
	if err != nil {
		logger.Debug("timeout waiting for a slot")
		return nil, merry.Here(err).WithValue("server", server)
	}

	defer c.limiter.Leave(ctx, server)

	logger.Debug("got slot for server",
		zap.String("name", server),
	)

	if r != nil {
		logger = logger.With(zap.Any("payloadData", r.LogInfo()))
	}
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		logger.Debug("error fetching result",
			zap.Error(err),
		)

		return nil, requestError(err, server)

	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// we don't need to process any further if the response is empty.
	if resp.StatusCode == http.StatusNotFound {
		return &ServerResponse{Server: server}, nil
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.Debug("error reading body",
			zap.Error(err),
		)
		return nil, merry.Here(err).WithValue("server", server)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, types.ErrFailedToFetch.WithValue("server", server).WithMessage(string(body)).WithHTTPCode(resp.StatusCode)
	}

	return &ServerResponse{Server: server, Response: body}, nil
}

func (c *HttpQuery) DoQuery(ctx context.Context, logger *zap.Logger, uri string, r types.Request) (*ServerResponse, merry.Error) {
	// not needed, can produce overload on busy cluster
	// if len(c.servers) > maxTries {
	// 	maxTries = len(c.servers)
	// }

	e := types.ErrFailedToFetch.WithValue("uri", uri)
	code := http.StatusInternalServerError
	var idx uint64 = math.MaxUint64
	var server string
	for try := 0; try < c.maxTries; try++ {
		server, idx = c.pickServer(idx, logger)
		res, err := c.doRequest(ctx, logger, server, uri, r)
		if err != nil {
			logger.Debug("have errors",
				zap.Error(err),
			)

			e = e.WithCause(err).WithHTTPCode(merry.HTTPCode(err))
			code = merry.HTTPCode(err)
			if c.maxTries > 1 {
				if _, ok := c.retryCodes[code]; ok {
					continue
				}
				break
			}
		}

		return res, nil
	}

	return nil, types.ErrMaxTriesExceeded.WithCause(e).WithHTTPCode(code)
}

func (c *HttpQuery) DoQueryToAll(ctx context.Context, logger *zap.Logger, uri string, r types.Request) ([]*ServerResponse, merry.Error) {
	maxTries := c.maxTries
	if len(c.servers) > maxTries {
		maxTries = len(c.servers)
	}

	res := make([]*ServerResponse, len(c.servers))
	e := types.ErrFailedToFetch.WithValue("uri", uri)
	responseCount := 0
	code := http.StatusInternalServerError
	for i := range c.servers {
		for try := 0; try < maxTries; try++ {
			response, err := c.doRequest(ctx, logger, c.servers[i], uri, r)
			if err != nil {
				logger.Debug("have errors",
					zap.Error(err),
				)

				e = e.WithCause(err)
				code = merry.HTTPCode(err)
				if c.maxTries > 1 {
					if _, ok := c.retryCodes[code]; ok {
						continue
					}
					break
				}
			}

			res[i] = response
			responseCount++
			break
		}
	}

	if responseCount == len(c.servers) {
		return res, nil
	}

	return res, types.ErrMaxTriesExceeded.WithCause(e).WithHTTPCode(code)
}
