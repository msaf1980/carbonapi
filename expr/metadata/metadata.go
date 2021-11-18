package metadata

import (
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/lomik/zapwriter"
	"go.uber.org/zap"
)

// RegisterRewriteFunctionWithFilename registers function for a rewrite phase in metadata and fills out all Description structs
func RegisterRewriteFunctionWithFilename(name, filename string, function parser.RewriteFunction) {
	parser.FunctionMD.Lock()
	defer parser.FunctionMD.Unlock()
	function.SetEvaluator(parser.FunctionMD.Evaluator)
	if _, ok := parser.FunctionMD.RewriteFunctions[name]; ok {
		n := parser.FunctionMD.RewriteFunctionsFilenames[name]
		logger := zapwriter.Logger("registerRewriteFunction")
		logger.Warn("function already registered, will register new anyway",
			zap.String("name", name),
			zap.String("current_filename", filename),
			zap.Strings("previous_filenames", n),
			zap.Stack("stack"),
		)
	} else {
		parser.FunctionMD.RewriteFunctionsFilenames[name] = make([]string, 0)
	}
	// Check if we are colliding with non-rewrite Functions
	if _, ok := parser.FunctionMD.Functions[name]; ok {
		n := parser.FunctionMD.FunctionsFilenames[name]
		logger := zapwriter.Logger("registerRewriteFunction")
		logger.Warn("non-rewrite function with the same name already registered",
			zap.String("name", name),
			zap.String("current_filename", filename),
			zap.Strings("previous_filenames", n),
			zap.Stack("stack"),
		)
	}
	parser.FunctionMD.RewriteFunctionsFilenames[name] = append(parser.FunctionMD.RewriteFunctionsFilenames[name], filename)
	parser.FunctionMD.RewriteFunctions[name] = function

	for k, v := range function.Description() {
		parser.FunctionMD.Descriptions[k] = v
		if _, ok := parser.FunctionMD.DescriptionsGrouped[v.Group]; !ok {
			parser.FunctionMD.DescriptionsGrouped[v.Group] = make(map[string]types.FunctionDescription)
		}
		parser.FunctionMD.DescriptionsGrouped[v.Group][k] = v
	}
}

// RegisterRewriteFunction registers function for a rewrite phase in metadata and fills out all Description structs
func RegisterRewriteFunction(name string, function parser.RewriteFunction) {
	RegisterRewriteFunctionWithFilename(name, "", function)
}

// RegisterFunctionWithFilename registers function in metadata and fills out all Description structs
func RegisterFunctionWithFilename(name, filename string, function parser.Function) {
	parser.FunctionMD.Lock()
	defer parser.FunctionMD.Unlock()
	function.SetEvaluator(parser.FunctionMD.Evaluator)

	if _, ok := parser.FunctionMD.Functions[name]; ok {
		n := parser.FunctionMD.FunctionsFilenames[name]
		logger := zapwriter.Logger("registerFunction")
		logger.Warn("function already registered, will register new anyway",
			zap.String("name", name),
			zap.String("current_filename", filename),
			zap.Strings("previous_filenames", n),
			zap.Stack("stack"),
		)
	} else {
		parser.FunctionMD.FunctionsFilenames[name] = make([]string, 0)
	}
	// Check if we are colliding with non-rewrite Functions
	if _, ok := parser.FunctionMD.RewriteFunctions[name]; ok {
		n := parser.FunctionMD.RewriteFunctionsFilenames[name]
		logger := zapwriter.Logger("registerRewriteFunction")
		logger.Warn("rewrite function with the same name already registered",
			zap.String("name", name),
			zap.String("current_filename", filename),
			zap.Strings("previous_filenames", n),
			zap.Stack("stack"),
		)
	}
	parser.FunctionMD.Functions[name] = function
	parser.FunctionMD.FunctionsFilenames[name] = append(parser.FunctionMD.FunctionsFilenames[name], filename)

	for k, v := range function.Description() {
		parser.FunctionMD.Descriptions[k] = v
		if _, ok := parser.FunctionMD.DescriptionsGrouped[v.Group]; !ok {
			parser.FunctionMD.DescriptionsGrouped[v.Group] = make(map[string]types.FunctionDescription)
		}
		parser.FunctionMD.DescriptionsGrouped[v.Group][k] = v
	}
}

// RegisterFunction registers function in metadata and fills out all Description structs
func RegisterFunction(name string, function parser.Function) {
	RegisterFunctionWithFilename(name, "", function)
}

// SetEvaluator sets new evaluator function to be default for everything that needs it
func SetEvaluator(evaluator parser.Evaluator) {
	parser.FunctionMD.Lock()
	defer parser.FunctionMD.Unlock()

	parser.FunctionMD.Evaluator = evaluator
	for _, v := range parser.FunctionMD.Functions {
		v.SetEvaluator(evaluator)
	}

	for _, v := range parser.FunctionMD.RewriteFunctions {
		v.SetEvaluator(evaluator)
	}
}

// GetEvaluator returns evaluator
func GetEvaluator() parser.Evaluator {
	parser.FunctionMD.RLock()
	defer parser.FunctionMD.RUnlock()

	return parser.FunctionMD.Evaluator
}
