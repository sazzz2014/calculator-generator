// Run from the repository root after docker compose build.
package main

import (
	"context"
	"fmt"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	project := "nativecalc-smoke-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	env := append(os.Environ(), "HTTP_PORT="+strconv.Itoa(port), "GENERATOR_WORKERS=4")
	compose := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", project}, args...)...)
		cmd.Env = env
		out, e := cmd.CombinedOutput()
		if e != nil {
			return string(out), fmt.Errorf("docker compose %v: %w\n%s", args, e, out)
		}
		return string(out), nil
	}
	defer func() {
		out, e := compose("down", "--volumes")
		if e != nil {
			fmt.Fprintln(os.Stderr, out, e)
		}
	}()
	if _, err = compose("up", "-d", "--wait", "calculator"); err != nil {
		return err
	}
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: 3 * time.Second}
	for _, n := range []int{10, -3, 5} {
		resp, e := client.Post(base+"/calc?num="+strconv.Itoa(n), "", nil)
		if e != nil {
			return e
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("POST: %s", resp.Status)
		}
	}
	checkMetrics := func(loaded bool) error {
		resp, e := client.Get(base + "/metrics")
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("metrics: %s", resp.Status)
		}
		parser := expfmt.NewTextParser(model.LegacyValidation)
		families, e := parser.TextToMetricFamilies(resp.Body)
		if e != nil {
			return e
		}
		rps := families["calculator_http_rps"]
		if rps == nil || len(rps.Metric) != 60 {
			return fmt.Errorf("expected 60 RPS series")
		}
		calls := families["calculator_native_call_duration_seconds"]
		if calls == nil || len(calls.Metric) != 2 {
			return fmt.Errorf("missing native call metrics")
		}
		for _, m := range calls.Metric {
			if len(m.GetSummary().Quantile) != 2 || m.GetSummary().GetSampleCount() == 0 {
				return fmt.Errorf("missing native observations")
			}
		}
		if loaded {
			var total float64
			for _, m := range rps.Metric {
				total += m.GetGauge().GetValue()
			}
			if total == 0 {
				return fmt.Errorf("no generator RPS")
			}
		}
		return nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		err = checkMetrics(false)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err = compose("stop", "calculator"); err != nil {
		return err
	}
	logs, err := compose("logs", "--no-color", "calculator")
	if err != nil {
		return err
	}
	if !strings.Contains(logs, "event=calculation_final sum=12 sub=-12") {
		return fmt.Errorf("incorrect SIGINT final state:\n%s", logs)
	}
	if strings.Count(logs, "event=calculation_final") != 1 {
		return fmt.Errorf("expected one final record")
	}
	if _, err = compose("up", "-d", "--wait", "--force-recreate"); err != nil {
		return err
	}
	deadline = time.Now().Add(10 * time.Second)
	for {
		err = checkMetrics(true)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err = compose("stop", "generator"); err != nil {
		return err
	}
	if _, err = compose("stop", "calculator"); err != nil {
		return err
	}
	logs, err = compose("logs", "--no-color", "calculator")
	if err != nil {
		return err
	}
	matches := regexp.MustCompile(`event=calculation_final sum=(-?\d+) sub=(-?\d+)`).FindAllStringSubmatch(logs, -1)
	if len(matches) != 1 {
		return fmt.Errorf("missing final state: %s", logs)
	}
	sum, _ := strconv.ParseInt(matches[0][1], 10, 64)
	sub, _ := strconv.ParseInt(matches[0][2], 10, 64)
	if sum != -sub {
		return fmt.Errorf("inconsistent final pair")
	}
	fmt.Println("Smoke passed: native calls, Prometheus metrics, generator RPS and SIGINT drain.")
	return nil
}
