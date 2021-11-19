package expr

import (
	"context"

	utilctx "github.com/go-graphite/carbonapi/util/ctx"
	"github.com/msaf1980/go-stringutils"

	"github.com/ansel1/merry"
	"github.com/go-graphite/carbonapi/cmd/carbonapi/config"
	"github.com/go-graphite/carbonapi/expr/helper"
	_ "github.com/go-graphite/carbonapi/expr/interfaces"
	"github.com/go-graphite/carbonapi/expr/metadata"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
)

type evaluator struct{}

func metricPathWithFilter(path string, filter []*pb.FilteringFunction, n int) string {
	if n < 0 || n > len(filter) {
		n = len(filter)
	}
	if n == 0 {
		return path
	}
	var sb stringutils.Builder
	sb.Grow(len(path) + len(filter)*48)
	sb.WriteString(path)
	for i := 0; i < n; i++ {
		sb.WriteString(" | ")
		sb.WriteString(filter[i].Name)
		sb.WriteByte('(')
		for j, arg := range filter[i].Arguments {
			if j == 0 {
				sb.WriteByte('\'')
			} else {
				sb.WriteString(",'")
			}
			sb.WriteString(arg)
			sb.WriteByte('\'')
		}
		sb.WriteByte(')')
	}

	return sb.String()
}

// FetchAndEvalExp fetch data and evalualtes expressions
func (eval evaluator) FetchAndEvalExp(ctx context.Context, exp parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error) {
	config.Config.Limiter.Enter()
	defer config.Config.Limiter.Leave()

	multiFetchRequest := pb.MultiFetchRequest{}
	metricRequestCache := make(map[string]parser.MetricRequest)
	maxDataPoints := utilctx.GetMaxDatapoints(ctx)
	// values related to this particular `target=`
	targetValues := make(map[parser.MetricRequest][]*types.MetricData)

	metrics, err := exp.Metrics()
	if err != nil {
		return nil, err
	}

	for _, m := range metrics {
		fetchRequest := pb.FetchRequest{
			Name:            m.Metric,
			PathExpression:  m.Metric,
			StartTime:       m.From + from,
			StopTime:        m.Until + until,
			MaxDataPoints:   maxDataPoints,
			FilterFunctions: m.Filter,
		}

		metricRequest := parser.MetricRequest{
			Metric: metricPathWithFilter(fetchRequest.PathExpression, m.Filter, -1),
			From:   fetchRequest.StartTime,
			Until:  fetchRequest.StopTime,
		}

		// avoid multiple requests in a function, E.g divideSeries(a.b, a.b)
		if m.Metric != metricRequest.Metric {
			if cachedMetricRequest, ok := metricRequestCache[metricRequest.Metric]; ok &&
				cachedMetricRequest.From == metricRequest.From &&
				cachedMetricRequest.Until == metricRequest.Until {
				continue
			}
		}

		// avoid multiple requests in a http request, E.g render?target=a.b&target=a.b
		if _, ok := values[metricRequest]; ok {
			targetValues[metricRequest] = nil
			continue
		}

		// avoid multiple requests from the same target, e.g. target=max(a,asPercent(holtWintersForecast(a),a))
		if _, ok := targetValues[metricRequest]; ok {
			continue
		}

		metricRequestCache[metricRequest.Metric] = metricRequest
		targetValues[metricRequest] = nil
		multiFetchRequest.Metrics = append(multiFetchRequest.Metrics, fetchRequest)
	}

	if len(multiFetchRequest.Metrics) > 0 {
		metrics, _, err := config.Config.ZipperInstance.Render(ctx, multiFetchRequest)
		// If we had only partial result, we want to do our best to actually do our job
		if err != nil && merry.HTTPCode(err) >= 400 && exp.Target() != "fallbackSeries" {
			return nil, err
		}
		for _, metric := range metrics {
			metricRequest := metricRequestCache[metric.PathExpression]
			if metric.RequestStartTime != 0 && metric.RequestStopTime != 0 {
				metricRequest.From = metric.RequestStartTime
				metricRequest.Until = metric.RequestStopTime
			}
			data, ok := values[metricRequest]
			if !ok {
				data = make([]*types.MetricData, 0, 1)
			}
			values[metricRequest] = append(data, metric)
		}
	}

	for m := range targetValues {
		targetValues[m] = values[m]
	}

	if config.Config.ZipperInstance.ScaleToCommonStep() {
		targetValues = helper.ScaleValuesToCommonStep(targetValues)
	}

	return eval.Eval(ctx, exp, from, until, targetValues)
}

// Eval evalualtes expressions
func (eval evaluator) Eval(ctx context.Context, exp parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) (results []*types.MetricData, err error) {
	rewritten, targets, err := RewriteExpr(ctx, exp, from, until, values)
	if err != nil {
		return nil, err
	}
	if rewritten {
		for _, target := range targets {
			exp, _, err = parser.ParseExpr(target)
			if err != nil {
				return nil, err
			}
			result, err := eval.FetchAndEvalExp(ctx, exp, from, until, values)
			if err != nil {
				return nil, err
			}
			results = append(results, result...)
		}
		return results, nil
	}
	return EvalExpr(ctx, exp, from, until, values)
}

var _evaluator = evaluator{}

func init() {
	helper.SetEvaluator(_evaluator)
	metadata.SetEvaluator(_evaluator)
}

// FetchAndEvalExp fetch data and evalualtes expressions
func FetchAndEvalExp(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error) {
	return _evaluator.FetchAndEvalExp(ctx, e, from, until, values)
}

// Eval is the main expression evaluator
func EvalExpr(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error) {
	if e.IsName() {
		return values[parser.MetricRequest{Metric: e.Target(), From: from, Until: until}], nil
	} else if e.IsConst() {
		p := types.MetricData{
			FetchResponse: pb.FetchResponse{
				Name:   e.Target(),
				Values: []float64{e.FloatValue()},
			},
			Tags: map[string]string{"name": e.Target()},
		}
		return []*types.MetricData{&p}, nil
	}
	// evaluate the function

	// all functions have arguments -- check we do too
	if len(e.Args()) == 0 {
		err := merry.WithMessagef(parser.ErrMissingArgument, "target=%s: %s", e.Target(), parser.ErrMissingArgument)
		return nil, merry.WithHTTPCode(err, 400)
	}

	parser.FunctionMD.RLock()
	f, ok := parser.FunctionMD.Functions[e.Target()]
	parser.FunctionMD.RUnlock()
	if ok {
		v, err := f.Do(ctx, e, from, until, values)
		if err != nil {
			err = merry.WithMessagef(err, "function=%s: %s", e.Target(), err.Error())
			if merry.Is(
				err,
				parser.ErrMissingExpr,
				parser.ErrMissingComma,
				parser.ErrMissingQuote,
				parser.ErrUnexpectedCharacter,
				parser.ErrBadType,
				parser.ErrMissingArgument,
				parser.ErrMissingTimeseries,
				parser.ErrSeriesDoesNotExist,
				parser.ErrUnknownTimeUnits,
			) {
				err = merry.WithHTTPCode(err, 400)
			}
		}
		return v, err
	}

	return nil, merry.WithHTTPCode(helper.ErrUnknownFunction(e.Target()), 400)
}

// RewriteExpr expands targets that use applyByNode into a new list of targets.
// eg:
// applyByNode(foo*, 1, "%") -> (true, ["foo1", "foo2"], nil)
// sumSeries(foo) -> (false, nil, nil)
// Assumes that applyByNode only appears as the outermost function.
func RewriteExpr(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) (bool, []string, error) {
	if e.IsFunc() {
		parser.FunctionMD.RLock()
		f, ok := parser.FunctionMD.RewriteFunctions[e.Target()]
		parser.FunctionMD.RUnlock()
		if ok {
			return f.Do(ctx, e, from, until, values)
		}
	}
	return false, nil, nil
}
