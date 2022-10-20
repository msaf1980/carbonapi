package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/go-graphite/carbonapi/tests"
)

func BenchmarkMarshalJSON(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalJSON(data, 1.0, false)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalJSONLong(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalJSON(data, 1.0, false)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalRaw(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalRaw(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalRawLong(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalRaw(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalCSV(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalCSV(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalCSVLong(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalCSV(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalPickle(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalPickle(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalPickleLong(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body := types.MarshalPickle(data)
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalProtobufV2(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body, err := types.MarshalProtobufV2(data)
		if err != nil {
			b.Fatal(err)
		}
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalProtobufV2Long(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body, err := types.MarshalProtobufV2(data)
		if err != nil {
			b.Fatal(err)
		}
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalProtobufV3(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body, err := types.MarshalProtobufV3(data)
		if err != nil {
			b.Fatal(err)
		}
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}

func BenchmarkMarshalProtobufV3Long(b *testing.B) {
	data := []*types.MetricData{
		types.MakeMetricData("metric1", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric2", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric3", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric4", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric5", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric6", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric7", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric8", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric9", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric10", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric11", tests.GetData(10000), 100, 100),
		types.MakeMetricData("metric12", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric13", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric14", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric15", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric16", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric17", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric18", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric19", tests.GetData(100000), 100, 100),
		types.MakeMetricData("metric20", tests.GetData(100000), 100, 100),
	}

	w := httptest.NewRecorder()
	w.Body.Grow(8196)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		body, err := types.MarshalProtobufV3(data)
		if err != nil {
			b.Fatal(err)
		}
		writeResponse(w, http.StatusOK, body, jsonFormat, "", "UUID")
	}
}
