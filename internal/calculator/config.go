package calculator

import (
	"example.com/nativecalculator/internal/config"
	"runtime"
	"time"
)

type Config struct {
	Addr                         string
	Workers, QueueSize           int
	LogInterval, ShutdownTimeout time.Duration
}

func LoadConfig() (c Config, err error) {
	if c.Addr, err = config.String("HTTP_ADDR", ":8080"); err != nil {
		return
	}
	if c.Workers, err = config.PositiveInt("CALC_WORKERS", runtime.NumCPU()); err != nil {
		return
	}
	if c.QueueSize, err = config.PositiveInt("CALC_QUEUE_SIZE", 4096); err != nil {
		return
	}
	if c.LogInterval, err = config.Duration("LOG_INTERVAL", 5*time.Second); err != nil {
		return
	}
	c.ShutdownTimeout, err = config.Duration("SHUTDOWN_TIMEOUT", 30*time.Second)
	return
}
