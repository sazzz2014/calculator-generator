package generator

import (
	"example.com/nativecalculator/internal/config"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	URL     string
	Workers int
	Timeout time.Duration
}

func LoadConfig() (c Config, err error) {
	if c.URL, err = config.String("CALCULATOR_URL", "http://calculator:8080"); err != nil {
		return
	}
	u, e := url.Parse(c.URL)
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return c, fmt.Errorf("CALCULATOR_URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	if u.Port() != "" {
		port, portErr := strconv.Atoi(u.Port())
		if portErr != nil || port < 1 || port > 65535 {
			return c, fmt.Errorf("CALCULATOR_URL port must be in 1..65535")
		}
	}
	if c.Workers, err = config.PositiveInt("GENERATOR_WORKERS", 8); err != nil {
		return
	}
	c.Timeout, err = config.Duration("HTTP_TIMEOUT", 10*time.Second)
	return
}
