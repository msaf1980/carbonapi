package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type extractTagTestCase struct {
	TestName string
	Input    string
	Output   map[string]string
}

func TestExtractTags(t *testing.T) {
	tests := []extractTagTestCase{
		{
			TestName: "NoTags",
			Input:    "metric",
			Output: map[string]string{
				"name": "metric",
			},
		},
		{
			TestName: "FewTags",
			Input:    "metricWithSomeTags;tag1=v1;tag2=v2;tag3=this is value with string",
			Output: map[string]string{
				"name": "metricWithSomeTags",
				"tag1": "v1",
				"tag2": "v2",
				"tag3": "this is value with string",
			},
		},
		{
			TestName: "BrokenTags",
			Input:    "metric;tag1=v1;;tag2=v2;tag3=;tag4;tag5=value=with=other=equal=signs;tag6=value=with-equal-signs-2",
			Output: map[string]string{
				"name": "metric",
				"tag1": "v1",
				"tag2": "v2",
				"tag3": "",
				"tag4": "",
				"tag5": "value=with=other=equal=signs",
				"tag6": "value=with-equal-signs-2",
			},
		},
		{
			TestName: "BrokenTags2",
			Input:    "metric;tag1=v1;",
			Output: map[string]string{
				"name": "metric",
				"tag1": "v1",
			},
		},
		{
			TestName: "BrokenTags2",
			Input:    "metric;tag1",
			Output: map[string]string{
				"name": "metric",
				"tag1": "",
			},
		},
		{
			TestName: "BrokenTags3",
			Input:    "metric;=;=",
			Output: map[string]string{
				"name": "metric",
			},
		},
		// from aggregation functions with seriesByTag
		{
			TestName: "seriesByTag('tag2=value*', 'name=metric')",
			Input:    "seriesByTag('tag2=value*', 'name=metric')",
			Output:   map[string]string{"name": "metric", "tag2": "value__"},
		},
		{
			TestName: "seriesByTag('tag2=~^value.*', 'name=metric')",
			Input:    "seriesByTag('tag2=~^value.*', 'name=metric')",
			Output:   map[string]string{"name": "metric", "tag2": "__value____"},
		},
		{
			TestName: "seriesByTag('tag2!=value21', 'name=metric')",
			Input:    "seriesByTag('tag2!=value21', 'name=metric')",
			Output:   map[string]string{"name": "metric", "tag2": "!value21"},
		},
		{
			TestName: "seriesByTag('tag2=value21')",
			Input:    "seriesByTag('tag2=value21')",
			Output:   map[string]string{"tag2": "value21"},
		},
		// brokken, from aggregation functions with seriesByTag
		{
			TestName: "seriesByTag('tag2=', 'name=metric')",
			Input:    "seriesByTag('tag2=', 'tag3', 'name=metric')",
			Output:   map[string]string{"name": "metric"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			res := ExtractTags(tt.Input)

			if len(res) != len(tt.Output) {
				t.Fatalf("result length mismatch, got %v, expected %v, %+v != %+v", len(res), len(tt.Output), res, tt.Output)
			}

			for k, v := range res {
				if expectedValue, ok := tt.Output[k]; ok {
					if v != expectedValue {
						t.Fatalf("value mismatch for key '%v': got '%v', exepcted '%v'", k, v, expectedValue)
					}
				} else {
					t.Fatalf("got unexpected key %v=%v in result", k, v)
				}
			}
		})
	}
}

func TestReplaceSpecSymbols(t *testing.T) {
	tests := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "_all__symbols__a-z____-___a_b___not_replaced_",
			Output: "_all__symbols__a-z____-___a_b___not_replaced_",
		},
		{
			Input:  "^some_symbols_[a-z]+.?-.*{a,b}?_are_replaced$",
			Output: "__some_symbols___a-z________-______a__b_____are_replaced__",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Input, func(t *testing.T) {
			out := replaceSpecSymbols(tt.Input)
			assert.Equal(t, tt.Output, out)
		})
	}
}

func BenchmarkReplaceSpecSymbols(b *testing.B) {
	benchmarks := []string{
		"_all__symbols__a-z____-___a_b___not_replaced_",
		"^some_symbols_[a-z]+.?-.*{a,b}?_are_replaced$",
	}
	for _, bm := range benchmarks {
		b.Run(bm, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := replaceSpecSymbols(bm)
				_ = s
			}
		})
	}
}
