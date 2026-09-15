package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func String(key, fallback string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	if v == "" {
		return "", fmt.Errorf("%s must not be empty", key)
	}
	return v, nil
}

func PositiveInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

func Duration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return d, nil
}
