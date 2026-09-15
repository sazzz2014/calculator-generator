package calculator

import (
	"bytes"
	"context"
	"example.com/nativecalculator/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type fakeNative struct{}

func (fakeNative) Add(a, b int64) int64 { return a + b }
func (fakeNative) Sub(a, b int64) int64 { return a - b }
func newMetrics() *metrics.Metrics      { return metrics.New(prometheus.NewRegistry(), time.Now) }

func TestHTTPValidation(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		code         int
	}{
		{"POST", "/calc?num=10", 200}, {"POST", "/calc?num=-9223372036854775808", 200},
		{"POST", "/calc", 400}, {"POST", "/calc?num=", 400}, {"POST", "/calc?num=x", 400},
		{"POST", "/calc?num=9223372036854775808", 400}, {"POST", "/calc?num=1&num=2", 400},
		{"POST", "/calc?num=1&bad=%zz", 400}, {"GET", "/calc?num=1", 405},
		{"GET", "/healthz", 200}, {"POST", "/healthz", 405}, {"POST", "/metrics", 405}, {"GET", "/unknown", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			m := metrics.New(reg, time.Now)
			q := make(chan int64, 1)
			gate, h := NewHandler(q, m, reg)
			defer gate.Stop()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.code {
				t.Fatalf("got %d want %d", w.Code, tc.code)
			}
			if tc.code == 200 && tc.method == "POST" {
				if len(q) != 1 {
					t.Fatal("not enqueued")
				}
			}
			if tc.code == 405 && w.Header().Get("Allow") == "" {
				t.Fatal("missing Allow")
			}
		})
	}
}

func TestFullQueueCancellationAndStop(t *testing.T) {
	for _, stop := range []bool{false, true} {
		reg := prometheus.NewRegistry()
		m := metrics.New(reg, time.Now)
		q := make(chan int64, 1)
		q <- 1
		gate, h := NewHandler(q, m, reg)
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest("POST", "/calc?num=2", nil).WithContext(ctx)
		done := make(chan struct{})
		w := httptest.NewRecorder()
		go func() { h.ServeHTTP(w, req); close(done) }()
		if stop {
			gate.Stop()
		} else {
			cancel()
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handler stuck")
		}
		gate.Stop()
		gate.Wait()
		cancel()
		if w.Code != 503 || len(q) != 1 {
			t.Fatalf("code=%d len=%d", w.Code, len(q))
		}
	}
}

func TestConcurrentDrainAndSnapshots(t *testing.T) {
	p := NewProcessor(4, 8, fakeNative{}, newMetrics())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				p.queue <- 10
				p.queue <- -3
				p.queue <- 5
				sum, sub := p.Snapshot()
				if sum != -sub {
					t.Error("inconsistent pair")
				}
			}
		}()
	}
	wg.Wait()
	p.Drain()
	sum, sub := p.Snapshot()
	if sum != 9600 || sub != -9600 {
		t.Fatalf("%d %d", sum, sub)
	}
}

func TestRunShutdown(t *testing.T) {
	// Reserve a port, then retry startup until Run binds it.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output bytes.Buffer
	log := slog.New(slog.NewTextHandler(&output, nil))
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{Addr: addr, Workers: 2, QueueSize: 8, LogInterval: time.Hour, ShutdownTimeout: time.Second}, fakeNative{}, log)
	}()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, e := client.Get("http://" + addr + "/healthz")
		if e == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(e)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, n := range []string{"10", "-3", "5"} {
		resp, e := client.Post("http://"+addr+"/calc?num="+n, "", nil)
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatal(resp.Status)
		}
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown stuck")
	}
	if !bytes.Contains(output.Bytes(), []byte("event=calculation_final sum=12 sub=-12")) {
		t.Fatal(output.String())
	}
}
