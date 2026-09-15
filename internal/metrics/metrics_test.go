package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"strings"
	"testing"
	"time"
)

func TestWindowRollover(t *testing.T) {
	now := time.Unix(1000, 0)
	w := NewRPSWindow(func() time.Time { return now })
	w.Increment()
	w.Increment()
	if w.Snapshot()[0] != 0 {
		t.Fatal("current second included")
	}
	now = now.Add(60 * time.Second)
	w.Increment()
	if w.Snapshot()[59] != 2 {
		t.Fatal("oldest complete second overwritten")
	}
	if w.Snapshot()[0] != 0 {
		t.Fatal("gap not zero")
	}
	now = now.Add(time.Second)
	snap := w.Snapshot()
	if snap[0] != 1 || snap[59] != 0 {
		t.Fatal(snap)
	}
	now = now.Add(100 * time.Second)
	if w.Snapshot() != ([60]uint64{}) {
		t.Fatal("expired data")
	}
}

func TestPrometheusFormat(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg, time.Now)
	m.Accept()
	m.C.Observe(0.01)
	m.Rust.Observe(0.02)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, f := range families {
		if _, err := expfmt.MetricFamilyToText(&text, f); err != nil {
			t.Fatal(err)
		}
	}
	parser := expfmt.NewTextParser(model.LegacyValidation)
	parsed, err := parser.TextToMetricFamilies(strings.NewReader(text.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed["calculator_http_rps"].Metric) != 60 {
		t.Fatal("need 60 RPS series")
	}
	calls := parsed["calculator_native_call_duration_seconds"].Metric
	if len(calls) != 2 {
		t.Fatal("need two libraries")
	}
	for _, metric := range calls {
		q := metric.GetSummary().Quantile
		if len(q) != 2 || q[0].GetQuantile() != 0.95 || q[1].GetQuantile() != 0.99 {
			t.Fatal("wrong quantiles")
		}
	}
	if parsed["calculator_http_requests_total"].Metric[0].GetCounter().GetValue() != 1 {
		t.Fatal("wrong counter")
	}
}
