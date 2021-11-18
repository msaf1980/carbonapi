package expr

import (
	"reflect"
	"testing"

	_ "github.com/go-graphite/carbonapi/expr/interfaces"
	"github.com/go-graphite/carbonapi/pkg/parser"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
)

// Tests for (e *expr) metrics(filterChain []*pb.FilteringFunction)
// pkg/parser/parser.go

func TestExprMetrics(t *testing.T) {
	//set backend filtering for exclude and average function
	parser.FunctionMD.RLock()
	for _, function := range []string{"exclude", "average"} {
		if f, ok := parser.FunctionMD.Functions[function]; ok {
			f.SetBackendFiltered()
		}
	}
	parser.FunctionMD.RUnlock()

	tests := []struct {
		target  string
		want    []parser.MetricRequestWithFilter
		wantErr bool
	}{
		{
			target: "scaleToSeconds(exclude(test.value.*,'RejectedByFilter|SomeRequestsFailed'),60)",
			want: []parser.MetricRequestWithFilter{
				{
					Metric: "test.value.*",
					Filter: []*pb.FilteringFunction{
						{
							Name:      "exclude",
							Arguments: []string{"RejectedByFilter|SomeRequestsFailed"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			target: "divideSeries(exclude(test.*.cur,'RejectedByFilter|SomeRequestsFailed'),exclude(test.*.max,'RejectedByFilter|SomeRequestsFailed'))",
			want: []parser.MetricRequestWithFilter{
				{
					Metric: "test.*.cur",
					Filter: []*pb.FilteringFunction{
						{
							Name:      "exclude",
							Arguments: []string{"RejectedByFilter|SomeRequestsFailed"},
						},
					},
				},
				{
					Metric: "test.*.max",
					Filter: []*pb.FilteringFunction{
						{
							Name:      "exclude",
							Arguments: []string{"RejectedByFilter|SomeRequestsFailed"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			target: "sumSeries(exclude(test.*.cur,'RejectedByFilter|SomeRequestsFailed'))",
			want: []parser.MetricRequestWithFilter{
				{
					Metric: "test.*.cur",
					Filter: []*pb.FilteringFunction{
						{
							Name:      "exclude",
							Arguments: []string{"RejectedByFilter|SomeRequestsFailed"},
						},
						{
							Name: "sumSeries",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			exp, _, err := parser.ParseExpr(tt.target)
			if err != nil {
				t.Errorf("parser.ParseExpr() error = %v", err)
				return
			}
			got, err := exp.Metrics()
			if err != nil {
				t.Errorf("Expr.Metrics() error = %v", err)
				return
			}
			maxLen := len(got)
			if maxLen < len(tt.want) {
				maxLen = len(tt.want)
			}
			for i := 0; i < maxLen; i++ {
				if i >= len(got) {
					t.Errorf("+ expr.Metrics()[%d].Metric = %s", i, tt.want[i].Metric)
				} else if i >= len(tt.want) {
					t.Errorf("- expr.Metrics()[%d].Metric = %s", i, got[i].Metric)
				} else {
					if got[i].Metric != tt.want[i].Metric {
						t.Errorf("- expr.Metrics()[%d].Metric = %s, want %s", i, got[i].Metric, tt.want[i].Metric)
					}
					maxLenF := len(got[i].Filter)
					if maxLenF < len(tt.want[i].Filter) {
						maxLenF = len(tt.want[i].Filter)
					}
					for j := 0; j < maxLenF; j++ {
						if j >= len(got[i].Filter) {
							t.Errorf("- expr.Metrics()[%d].Filter[%d] = %+v", i, j, tt.want[i].Filter[j])
						} else if j >= len(tt.want[i].Filter) {
							t.Errorf("+ expr.Metrics()[%d].Filter[%d] = %+v", i, j, got[i].Filter[j])
						} else if !reflect.DeepEqual(got[i].Filter[j], tt.want[i].Filter[j]) {
							t.Errorf("- expr.Metrics()[%d].Filter[%d] = %+v", i, j, tt.want[i].Filter[j])
							t.Errorf("+ expr.Metrics()[%d].Filter[%d] = %+v", i, j, got[i].Filter[j])
						}
					}
				}
			}
		})
	}
}
