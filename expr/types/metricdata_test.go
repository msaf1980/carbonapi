package types

import (
	"bytes"
	"math"
	"testing"
)

func TestJSONResponse(t *testing.T) {

	tests := []struct {
		results []*MetricData
		out     []byte
	}{
		{
			[]*MetricData{
				MakeMetricData("metric1", []float64{1, 1.5, 2.25, math.NaN()}, 100, 100),
				MakeMetricData("metric2;foo=bar", []float64{2, 2.5, 3.25, 4, 5}, 100, 100),
			},
			[]byte(`[{"target":"metric1","datapoints":[[1,100],[1.5,200],[2.25,300],[null,400]],"tags":{"name":"metric1"}},{"target":"metric2;foo=bar","datapoints":[[2,100],[2.5,200],[3.25,300],[4,400],[5,500]],"tags":{"foo":"bar","name":"metric2"}}]`),
		},
	}

	for _, tt := range tests {
		b := MarshalJSON(tt.results, 1.0, false)
		if !bytes.Equal(b, tt.out) {
			t.Errorf("marshalJSON(%+v): got\n%+v\nwant\n%+v", tt.results, string(b), string(tt.out))
		}
	}
}

func TestJSONResponseNoNullPoints(t *testing.T) {

	tests := []struct {
		results []*MetricData
		out     []byte
	}{
		{
			[]*MetricData{
				MakeMetricData("metric1", []float64{1, 1.5, 2.25, math.NaN()}, 100, 100),
				MakeMetricData("metric2;foo=bar", []float64{math.NaN(), 2.5, 3.25, 4, 5}, 100, 100),
				MakeMetricData("metric3;foo=bar", []float64{2, math.NaN(), 3.25, 4, 5}, 100, 100),
				MakeMetricData("metric4;foo=bar", []float64{math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()}, 100, 100),
			},
			[]byte(`[{"target":"metric1","datapoints":[[1,100],[1.5,200],[2.25,300]],"tags":{"name":"metric1"}},{"target":"metric2;foo=bar","datapoints":[[2.5,200],[3.25,300],[4,400],[5,500]],"tags":{"foo":"bar","name":"metric2"}},{"target":"metric3;foo=bar","datapoints":[[2,100],[3.25,300],[4,400],[5,500]],"tags":{"foo":"bar","name":"metric3"}},{"target":"metric4;foo=bar","datapoints":[],"tags":{"foo":"bar","name":"metric4"}}]`),
		},
	}

	for _, tt := range tests {
		b := MarshalJSON(tt.results, 1.0, true)
		if !bytes.Equal(b, tt.out) {
			t.Errorf("marshalJSON(%+v): got\n%+v\nwant\n%+v", tt.results, string(b), string(tt.out))
		}
	}
}

func TestRawResponse(t *testing.T) {

	tests := []struct {
		results []*MetricData
		out     []byte
	}{
		{
			[]*MetricData{
				MakeMetricData("metric1", []float64{1, 1.5, 2.25, math.NaN()}, 100, 100),
				MakeMetricData("metric2", []float64{2, 2.5, 3.25, 4, 5}, 100, 100),
			},
			[]byte(`metric1,100,500,100|1,1.5,2.25,None` + "\n" + `metric2,100,600,100|2,2.5,3.25,4,5` + "\n"),
		},
	}

	for _, tt := range tests {
		b := MarshalRaw(tt.results)
		if !bytes.Equal(b, tt.out) {
			t.Errorf("marshalRaw(%+v): got\n%+v\nwant\n%+v", tt.results, string(b), string(tt.out))
		}
	}
}

func TestCSVResponse(t *testing.T) {

	tests := []struct {
		results []*MetricData
		out     []byte
	}{
		{
			[]*MetricData{
				MakeMetricData("metric1", []float64{1, 1.5, 2.25, math.NaN()}, 100, 100),
				MakeMetricData("metric2", []float64{2, 2.5, 3.25, 4, 5}, 100, 100),
			},
			[]byte(`"metric1",1970-01-01 00:01:40,1` + "\n" +
				`"metric1",1970-01-01 00:03:20,1.5` + "\n" +
				`"metric1",1970-01-01 00:05:00,2.25` + "\n" +
				`"metric1",1970-01-01 00:06:40,` + "\n" +
				`"metric2",1970-01-01 00:01:40,2` + "\n" +
				`"metric2",1970-01-01 00:03:20,2.5` + "\n" +
				`"metric2",1970-01-01 00:05:00,3.25` + "\n" +
				`"metric2",1970-01-01 00:06:40,4` + "\n" +
				`"metric2",1970-01-01 00:08:20,5` + "\n",
			),
		},
	}

	for _, tt := range tests {
		b := MarshalCSV(tt.results)
		if !bytes.Equal(b, tt.out) {
			t.Errorf("marshalCSV(%+v): \n%+v\nwant\n%+v", tt.results, string(b), string(tt.out))
		}
	}
}
