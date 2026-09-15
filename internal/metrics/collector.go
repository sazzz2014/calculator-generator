package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type Metrics struct {
	Window   *RPSWindow
	Accepted prometheus.Counter
	C, Rust  prometheus.Observer
	rps      *prometheus.Desc
}

func New(reg *prometheus.Registry, now func() time.Time) *Metrics {
	summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "calculator_native_call_duration_seconds",
		Help:       "Native FFI call duration; approximate quantiles over a 60 second window.",
		Objectives: map[float64]float64{0.95: 0.005, 0.99: 0.001},
		MaxAge:     time.Minute, AgeBuckets: 6,
	}, []string{"library"})
	m := &Metrics{
		Window: NewRPSWindow(now),
		Accepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "calculator_http_requests_total", Help: "Successfully accepted calculation requests.",
			ConstLabels: prometheus.Labels{"code": "200", "route": "/calc"},
		}),
		C: summary.WithLabelValues("c"), Rust: summary.WithLabelValues("rust"),
		rps: prometheus.NewDesc("calculator_http_rps", "Accepted requests in each of the last 60 complete seconds.", []string{"seconds_ago"}, nil),
	}
	reg.MustRegister(m, m.Accepted, summary)
	return m
}
func (m *Metrics) Accept()                             { m.Window.Increment(); m.Accepted.Inc() }
func (m *Metrics) Describe(ch chan<- *prometheus.Desc) { ch <- m.rps }
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	for i, count := range m.Window.Snapshot() {
		ch <- prometheus.MustNewConstMetric(m.rps, prometheus.GaugeValue, float64(count), strconv.Itoa(i+1))
	}
}
