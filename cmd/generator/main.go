package main

import (
	"context"
	"example.com/nativecalculator/internal/generator"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := generator.LoadConfig()
	if err != nil {
		log.Error("generator_failed", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	generator.Run(ctx, cfg, log)
}
