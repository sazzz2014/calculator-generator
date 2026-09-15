package calculator

import (
	"context"
	"example.com/nativecalculator/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// The gate makes WaitGroup.Add and shutdown mutually exclusive.
type Handler struct {
	mu       sync.Mutex
	stopping bool
	active   sync.WaitGroup
	stop     context.Context
	cancel   context.CancelFunc
	queue    chan<- int64
	metrics  *metrics.Metrics
}

func NewHandler(queue chan<- int64, m *metrics.Metrics, reg *prometheus.Registry) (*Handler, http.Handler) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Handler{stop: ctx, cancel: cancel, queue: queue, metrics: m}
	mux := http.NewServeMux()
	mux.HandleFunc("/calc", h.calculate)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		select {
		case <-ctx.Done():
			http.Error(w, "stopping", http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})
	prom := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if method(w, r, http.MethodGet) {
			prom.ServeHTTP(w, r)
		}
	})
	return h, mux
}
func method(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}
	w.Header().Set("Allow", want)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}
func (h *Handler) calculate(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "invalid query", http.StatusBadRequest)
		return
	}
	query := values["num"]
	if len(query) != 1 {
		http.Error(w, "exactly one num is required", http.StatusBadRequest)
		return
	}
	num, err := strconv.ParseInt(query[0], 10, 64)
	if err != nil {
		http.Error(w, "num must be int64", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	if h.stopping {
		h.mu.Unlock()
		http.Error(w, "stopping", http.StatusServiceUnavailable)
		return
	}
	h.active.Add(1)
	h.mu.Unlock()
	defer h.active.Done()
	select {
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusServiceUnavailable)
	case <-h.stop.Done():
		http.Error(w, "stopping", http.StatusServiceUnavailable)
	case h.queue <- num:
		h.metrics.Accept()
		w.WriteHeader(http.StatusOK)
	}
}
func (h *Handler) Stop() {
	h.mu.Lock()
	h.stopping = true
	h.cancel()
	h.mu.Unlock()
}
func (h *Handler) Wait() { h.active.Wait() }
