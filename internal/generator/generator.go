package generator

import (
	"context"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Stats struct{ Success, Errors uint64 }

func number(r *rand.Rand) int { return r.IntN(201) - 100 }

func Run(ctx context.Context, cfg Config, log *slog.Logger) Stats {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, MaxIdleConns: cfg.Workers, MaxIdleConnsPerHost: cfg.Workers, MaxConnsPerHost: cfg.Workers, IdleConnTimeout: 90 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: cfg.Timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	var success, failures atomic.Uint64
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		seed := rand.Uint64()
		wg.Add(1)
		go func(worker int, seed uint64) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(seed, uint64(worker)+1))
			var lastLog time.Time
			for ctx.Err() == nil {
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.URL, "/")+"/calc?num="+strconv.Itoa(number(rng)), nil)
				if err != nil {
					failures.Add(1)
					log.Error("generator_request_failed", "error", err)
					return
				}
				resp, err := client.Do(req)
				status := 0
				if resp != nil {
					status = resp.StatusCode
					_, readErr := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
					resp.Body.Close()
					if err == nil {
						err = readErr
					}
				}
				if ctx.Err() != nil {
					return
				}
				if err == nil && status == http.StatusOK {
					success.Add(1)
					continue
				}
				failures.Add(1)
				if time.Since(lastLog) >= time.Second {
					log.Warn("generator_request_failed", "worker", worker, "status", status, "error", err)
					lastLog = time.Now()
				}
				// Avoid burning a CPU when the server refuses connections immediately.
				timer := time.NewTimer(50 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}(i, seed)
	}
	log.Info("generator_started", "workers", cfg.Workers, "url", cfg.URL)
	wg.Wait()
	result := Stats{Success: success.Load(), Errors: failures.Load()}
	log.Info("generator_final", "success", result.Success, "errors", result.Errors)
	return result
}
