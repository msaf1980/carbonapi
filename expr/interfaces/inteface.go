package interfaces

import "github.com/go-graphite/carbonapi/pkg/parser"

// FunctionBase is a set of base methods that partly satisfy Function interface and most probably nobody will modify
type FunctionBase struct {
	Evaluator          parser.Evaluator
	canBackendFiltered bool // TODO: read from config
}

// SetEvaluator sets evaluator
func (b *FunctionBase) SetEvaluator(evaluator parser.Evaluator) {
	b.Evaluator = evaluator
}

// GetEvaluator returns evaluator
func (b *FunctionBase) GetEvaluator() parser.Evaluator {
	return b.Evaluator
}

func (b *FunctionBase) CanBackendFiltered() bool {
	return b.canBackendFiltered
}

func (b *FunctionBase) SetBackendFiltered() {
	b.canBackendFiltered = true
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
	F        parser.RewriteFunction
}

type FunctionMetadata struct {
	Name     string
	Filename string
	Order    Order
	F        parser.Function
}
