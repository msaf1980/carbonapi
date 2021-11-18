package interfaces

import (
	"context"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
)

// FunctionBase is a set of base methods that partly satisfy Function interface and most probably nobody will modify
type FunctionBase struct {
	Evaluator          Evaluator
	canBackendFiltered bool // TODO: read from config
}

// SetEvaluator sets evaluator
func (b *FunctionBase) SetEvaluator(evaluator Evaluator) {
	b.Evaluator = evaluator
}

// GetEvaluator returns evaluator
func (b *FunctionBase) GetEvaluator() Evaluator {
	return b.Evaluator
}

func (b *FunctionBase) CanBackendFiltered() bool {
	return b.canBackendFiltered
}

func (b *FunctionBase) SetBackendFiltered() {
	b.canBackendFiltered = true
}

// Evaluator is a interface for any existing expression parser
type Evaluator interface {
	Eval(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error)
}

type Order int

const (
	Any Order = iota
	Last
)

type RewriteFunctionMetadata struct {
	Name     string
	Filename string
	Order    Order
	F        RewriteFunction
}

type FunctionMetadata struct {
	Name     string
	Filename string
	Order    Order
	F        Function
}

// Function is interface that all graphite functions should follow
type Function interface {
	SetEvaluator(evaluator Evaluator)
	GetEvaluator() Evaluator
	Do(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) ([]*types.MetricData, error)
	Description() map[string]types.FunctionDescription
	SetBackendFiltered()
	CanBackendFiltered() bool
}

// Function is interface that all graphite functions should follow
type RewriteFunction interface {
	SetEvaluator(evaluator Evaluator)
	GetEvaluator() Evaluator
	Do(ctx context.Context, e parser.Expr, from, until int64, values map[parser.MetricRequest][]*types.MetricData) (bool, []string, error)
	Description() map[string]types.FunctionDescription
}
