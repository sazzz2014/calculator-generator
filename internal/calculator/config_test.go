package calculator

import "testing"

func TestConfig(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":8081")
	t.Setenv("CALC_WORKERS", "2")
	t.Setenv("CALC_QUEUE_SIZE", "4")
	t.Setenv("LOG_INTERVAL", "5s")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")
	cfg, err := LoadConfig()
	if err != nil || cfg.Addr != ":8081" || cfg.Workers != 2 || cfg.QueueSize != 4 {
		t.Fatal(cfg, err)
	}
	for _, key := range []string{"HTTP_ADDR", "CALC_WORKERS", "CALC_QUEUE_SIZE", "LOG_INTERVAL", "SHUTDOWN_TIMEOUT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "")
			if _, err := LoadConfig(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}
