package main

import (
	"context"
	"example.com/nativecalculator/internal/calculator"
	"example.com/nativecalculator/internal/native"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := calculator.LoadConfig()
	if err == nil {
		var lib native.Calculator
		lib, err = native.New()
		if err == nil {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			err = calculator.Run(ctx, cfg, lib, log)
		}
	}
	if err != nil {
		log.Error("calculator_failed", "error", err)
		os.Exit(1)
	}
}
