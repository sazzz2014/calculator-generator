package calculator

import (
	"context"
	"errors"
	"example.com/nativecalculator/internal/metrics"
	"example.com/nativecalculator/internal/native"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func Run(ctx context.Context, cfg Config, lib native.Calculator, log *slog.Logger) error {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	reg := prometheus.NewRegistry()
	m := metrics.New(reg, time.Now)
	p := NewProcessor(cfg.Workers, cfg.QueueSize, lib, m)
	admission, handler := NewHandler(p.queue, m, reg)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	reportCtx, stopReport := context.WithCancel(context.Background())
	reportDone := make(chan struct{})
	go func() {
		defer close(reportDone)
		ticker := time.NewTicker(cfg.LogInterval)
		defer ticker.Stop()
		for {
			select {
			case <-reportCtx.Done():
				return
			case <-ticker.C:
				sum, sub := p.Snapshot()
				log.Info("calculation_snapshot", "event", "calculation_snapshot", "sum", sum, "sub", sub)
			}
		}
	}()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	log.Info("calculator_started", "addr", listener.Addr().String(), "workers", cfg.Workers, "queue_size", cfg.QueueSize)
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-served:
	}
	admission.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("http_shutdown_timeout", "error", err)
		_ = server.Close()
	}
	cancel()
	admission.Wait()
	// Drain is deliberately not cut short by the HTTP shutdown deadline.
	p.Drain()
	stopReport()
	<-reportDone
	sum, sub := p.Snapshot()
	log.Info("calculation_final", "event", "calculation_final", "sum", sum, "sub", sub)
	if errors.Is(serveErr, http.ErrServerClosed) {
		return nil
	}
	return serveErr
}
