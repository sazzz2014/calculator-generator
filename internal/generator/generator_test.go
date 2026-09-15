package generator

import (
	"context"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestRange(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	seen := map[int]bool{}
	for i := 0; i < 10000; i++ {
		n := number(r)
		if n < -100 || n > 100 {
			t.Fatal(n)
		}
		seen[n] = true
	}
	if len(seen) != 201 {
		t.Fatal("range incomplete")
	}
}
func TestURLValidation(t *testing.T) {
	for _, u := range []string{"", "://bad", "ftp://server", "http:///", "http://user:pass@host", "http://host?x=y", "http://host/#fragment", "http://host:0"} {
		t.Setenv("CALCULATOR_URL", u)
		if _, err := LoadConfig(); err == nil {
			t.Fatal(u)
		}
	}
	t.Setenv("CALCULATOR_URL", "http://localhost:8080")
	if _, err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GENERATOR_WORKERS", "0")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("zero workers")
	}
}
func TestLoadAndReuse(t *testing.T) {
	var requests, connections, active, maxActive atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := active.Add(1)
		defer active.Add(-1)
		for old := maxActive.Load(); a > old; old = maxActive.Load() {
			if maxActive.CompareAndSwap(old, a) {
				break
			}
		}
		n, err := strconv.Atoi(r.URL.Query().Get("num"))
		if err != nil || n < -100 || n > 100 || r.Method != "POST" || r.URL.Path != "/calc" {
			t.Error("invalid request")
		}
		w.WriteHeader(200)
		if requests.Add(1) >= 100 {
			cancel()
		}
	}))
	server.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()
	watchdog := time.AfterFunc(5*time.Second, cancel)
	defer watchdog.Stop()
	stats := Run(ctx, Config{URL: server.URL, Workers: 4, Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if requests.Load() < 100 || stats.Success == 0 || stats.Errors != 0 {
		t.Fatalf("requests=%d stats=%+v", requests.Load(), stats)
	}
	if connections.Load() >= requests.Load()/2 || maxActive.Load() > 4 {
		t.Fatalf("connections=%d max=%d", connections.Load(), maxActive.Load())
	}
}
