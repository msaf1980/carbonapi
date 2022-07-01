package helper

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/go-graphite/carbonapi/expr/tags"
	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/pkg/parser"
)

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		expected map[string]string
	}{
		{
			name:   "tagged metric",
			metric: "cpu.usage_idle;cpu=cpu-total;host=test",
			expected: map[string]string{
				"name": "cpu.usage_idle",
				"cpu":  "cpu-total",
				"host": "test",
			},
		},
		{
			name:   "no tags in metric",
			metric: "cpu.usage_idle",
			expected: map[string]string{
				"name": "cpu.usage_idle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tags.ExtractTags(tt.metric)
			if len(actual) != len(tt.expected) {
				t.Fatalf("amount of tags doesn't match: got %v, expected %v", actual, tt.expected)
			}
			for tag, value := range actual {
				vExpected, ok := tt.expected[tag]
				if !ok {
					t.Fatalf("tag %v not found in %+v", value, actual)
				} else if vExpected != value {
					t.Errorf("unexpected tag-value, got %v, expected %v", value, vExpected)
				}
			}
		})
	}
}

func TestGCD(t *testing.T) {
	tests := []struct {
		arg1     int64
		arg2     int64
		expected int64
	}{
		{
			13,
			17,
			1,
		},
		{
			14,
			21,
			7,
		},
		{
			12,
			16,
			4,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("GDC(%v, %v)=>%v", tt.arg1, tt.arg2, tt.expected), func(t *testing.T) {
			value := GCD(tt.arg1, tt.arg2)
			if value != tt.expected {
				t.Errorf("GCD of %v and %v != %v: %v", tt.arg1, tt.arg2, tt.expected, value)
			}
		})
	}
}

func TestLCM(t *testing.T) {
	tests := []struct {
		args     []int64
		expected int64
	}{
		{
			[]int64{2, 3},
			6,
		},
		{
			[]int64{},
			0,
		},
		{
			[]int64{15},
			15,
		},
		{
			[]int64{10, 15, 20},
			60,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("LMC(%v)=>%v", tt.args, tt.expected), func(t *testing.T) {
			value := LCM(tt.args...)
			if value != tt.expected {
				t.Errorf("LCM of %v != %v: %v", tt.args, tt.expected, value)
			}
		})
	}
}

func TestGetCommonStep(t *testing.T) {
	tests := []struct {
		metrics    []*types.MetricData
		commonStep int64
		changed    bool
	}{
		// Different steps and start/stop time
		{
			[]*types.MetricData{
				types.MakeMetricData("metric1", make([]float64, 15), 5, 5), // 5..80
				types.MakeMetricData("metric2", make([]float64, 30), 2, 4), // 4..64
				types.MakeMetricData("metric2", make([]float64, 25), 3, 6), // 6..81
			},
			30,
			true,
		},
		// Same set of points
		{
			[]*types.MetricData{
				types.MakeMetricData("metric1", make([]float64, 15), 5, 5), // 5..80
				types.MakeMetricData("metric2", make([]float64, 15), 5, 5), // 5..80
				types.MakeMetricData("metric3", make([]float64, 15), 5, 5), // 5..80
			},
			5,
			false,
		},
		// Same step, different lengths
		{
			[]*types.MetricData{
				types.MakeMetricData("metric1", make([]float64, 5), 5, 15), // 15..40
				types.MakeMetricData("metric2", make([]float64, 8), 5, 30), // 30..70
				types.MakeMetricData("metric3", make([]float64, 4), 5, 35), // 35..55
			},
			5,
			false,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Set %v", i), func(t *testing.T) {
			com, changed := GetCommonStep(tt.metrics)
			if com != tt.commonStep {
				t.Errorf("Result of GetCommonStep: %v; expected is %v", com, tt.commonStep)
			}
			if changed != tt.changed {
				t.Errorf("GetCommonStep changed: %v; expected is %v", changed, tt.changed)
			}
		})
	}
}

func TestScaleToCommonStep(t *testing.T) {
	NaN := math.NaN()
	tests := []struct {
		name       string
		metrics    []*types.MetricData
		commonStep int64
		expected   []*types.MetricData
	}{
		{
			"Normal metrics",
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{1, 3, 5, 7, 9, 11, 13, 15, 17}, 1, 4), // 4..13
				types.MakeMetricData("metric2", []float64{1, 2, 3, 4, 5}, 2, 4),                 // 4..14
				types.MakeMetricData("metric3", []float64{1, 2, 3, 4, 5, 6}, 3, 3),              // 3..21
			},
			0,
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{2, 10, 17}, 6, 0),      // 0..18
				types.MakeMetricData("metric2", []float64{1, 3, 5}, 6, 0),        // 0..18
				types.MakeMetricData("metric3", []float64{1, 2.5, 4.5, 6}, 6, 0), // 0..24
			},
		},
		{
			"xFilesFactor and custom aggregate function",
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{NaN, 3, 5, 7, 9, 11, 13, 15, 17}, 1, 3), // 3..12
				types.MakeMetricData("metric2", []float64{1, 2, 3, 4, 5}, 2, 4),                   // 4..14
				types.MakeMetricData("metric3", []float64{1, 2, 3, 4, 5, 6}, 3, 3),                // 3..21
				types.MakeMetricData("metric6", []float64{1, 2, 3, 4, 5}, 6, 0),                   // 0..30
			},
			0,
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{NaN, 72}, 6, 0),        // 0..12
				types.MakeMetricData("metric2", []float64{NaN, 2, NaN}, 6, 0),    // 0..18
				types.MakeMetricData("metric3", []float64{NaN, 3, 5, NaN}, 6, 0), // 0..24
				types.MakeMetricData("metric6", []float64{1, 2, 3, 4, 5}, 6, 0),  // 0..30, unchanged
			},
		},
		{
			"Custom common step",
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{NaN, 3, 5, 7, 9, 11, 13, 15, 17}, 1, 3), // 3..12
				types.MakeMetricData("metric2", []float64{1, 2, 3, 4, 5}, 2, 4),                   // 4..14
				types.MakeMetricData("metric3", []float64{1, 2, 3, 4, 5, 6}, 3, 3),                // 3..21
				types.MakeMetricData("metric6", []float64{1, 2, 3, 4, 5}, 6, 0),                   // 0..30
			},
			12,
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{10}, 12, 0),          // 0..12
				types.MakeMetricData("metric2", []float64{2.5, 5}, 12, 0),      // 0..18
				types.MakeMetricData("metric3", []float64{2, 5}, 12, 0),        // 0..24
				types.MakeMetricData("metric6", []float64{1.5, 3.5, 5}, 12, 0), // 0..30, unchanged
			},
		},
	}
	custom := tests[1].metrics
	custom[0].ConsolidationFunc = "sum"
	custom[1].ConsolidationFunc = "min"
	custom[2].ConsolidationFunc = "max"
	custom[0].XFilesFactor = 0.45
	custom[1].XFilesFactor = 0.45
	custom[2].XFilesFactor = 0.51
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleToCommonStep(tt.metrics, tt.commonStep)
			if len(result) != len(tt.expected) {
				t.Errorf("Result has different length %v than expected %v", len(result), len(tt.expected))
			}
			for i, r := range result {
				e := tt.expected[i]
				if r.StartTime != e.StartTime {
					t.Errorf("result[%v].StartTime %v != expected[%v].StartTime %v", i, r.StartTime, i, e.StartTime)
				}
				if r.StopTime != e.StopTime {
					t.Errorf("result[%v].StopTime %v != expected[%v].StopTime %v", i, r.StopTime, i, e.StopTime)
				}
				if r.StepTime != e.StepTime {
					t.Errorf("result[%v].StepTime %v != expected[%v].StepTime %v", i, r.StepTime, i, e.StepTime)
				}
				if len(r.Values) != len(e.Values) {
					t.Fatalf("Values of result[%v] has the different length %v than expected %v\ngot %+v, want %+v", i, len(r.Values), len(e.Values), r.Values, e.Values)
				}
				for v, rv := range r.Values {
					ev := e.Values[v]
					if math.IsNaN(rv) != math.IsNaN(ev) {
						t.Errorf("One of result[%v][%v] is NaN, but not the second: result=%v, expected=%v", i, v, rv, ev)
					} else if !math.IsNaN(rv) && (rv != ev) {
						t.Errorf("result[%v][%v] %v != expected[%v][%v]: %v", i, v, rv, i, v, ev)
					}
				}
			}
		})
	}
}

func TestScaleToCommonStep0(t *testing.T) {
	NaN := math.NaN()
	tests := []struct {
		name       string
		metrics    []*types.MetricData
		commonStep int64
		expected   []*types.MetricData
	}{
		{
			"Custom common step",
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{NaN, 3, 5, 7, 9, 11, 13, 15}, 3, 3).
					SetConsolidationFunc("sum"), // 3..27
				types.MakeMetricData("metric2", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 4).
					SetConsolidationFunc("max"), // 3..23
				types.MakeMetricData("metric3", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 4).
					SetConsolidationFunc("min"), // 3..23
				types.MakeMetricData("metric4", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}, 2, 4).
					SetConsolidationFunc("avg"), // 3..23
			},
			0,
			[]*types.MetricData{
				types.MakeMetricData("metric1", []float64{math.NaN(), 8, 16, 24, 15}, 6, 0), // 0..30
				types.MakeMetricData("metric2", []float64{1, 4, 7, 9}, 6, 0),                // 0..24
				types.MakeMetricData("metric3", []float64{1, 2, 5, 8}, 6, 0),                // 0..24
				types.MakeMetricData("metric3", []float64{1, 3, 6, 8.5}, 6, 0),              // 0..24
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleToCommonStep(tt.metrics, tt.commonStep)
			if len(result) != len(tt.expected) {
				t.Errorf("Result has different length %v than expected %v", len(result), len(tt.expected))
			}
			for i, r := range result {
				e := tt.expected[i]
				if r.StartTime != e.StartTime {
					t.Errorf("result[%v].StartTime %v != expected[%v].StartTime %v", i, r.StartTime, i, e.StartTime)
				}
				if r.StopTime != e.StopTime {
					t.Errorf("result[%v].StopTime %v != expected[%v].StopTime %v", i, r.StopTime, i, e.StopTime)
				}
				if r.StepTime != e.StepTime {
					t.Errorf("result[%v].StepTime %v != expected[%v].StepTime %v", i, r.StepTime, i, e.StepTime)
				}
				if len(r.Values) != len(e.Values) {
					t.Fatalf("Values of result[%v] has the different length %v than expected %v\ngot %+v, want %+v", i, len(r.Values), len(e.Values), r.Values, e.Values)
				}
				for v, rv := range r.Values {
					ev := e.Values[v]
					if math.IsNaN(rv) != math.IsNaN(ev) {
						t.Errorf("One of result[%v][%v] is NaN, but not the second: result=%v, expected=%v", i, v, rv, ev)
					} else if !math.IsNaN(rv) && (rv != ev) {
						t.Errorf("result[%v][%v] %v != expected[%v][%v]: %v", i, v, rv, i, v, ev)
					}
				}
			}
		})
	}
}

func TestAggregateSeries(t *testing.T) {
	type args struct {
		e        parser.Expr
		args     []*types.MetricData
		function AggregateFunc
	}
	tests := []struct {
		name    string
		args    args
		want    []*types.MetricData
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AggregateSeries(tt.args.e, tt.args.args, tt.args.function)
			if (err != nil) != tt.wantErr {
				t.Errorf("AggregateSeries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AggregateSeries() = %v, want %v", got, tt.want)
			}
		})
	}
}
