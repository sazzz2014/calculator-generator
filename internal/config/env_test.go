package config

import (
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "abc", "9999999999999999999999"} {
		t.Setenv("TEST_INT", value)
		if _, err := PositiveInt("TEST_INT", 1); err == nil {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"", "0s", "-1s", "abc"} {
		t.Setenv("TEST_DURATION", value)
		if _, err := Duration("TEST_DURATION", time.Second); err == nil {
			t.Fatal(value)
		}
	}
	t.Setenv("TEST_INT", "3")
	if n, e := PositiveInt("TEST_INT", 1); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	t.Setenv("TEST_DURATION", "2s")
	if d, e := Duration("TEST_DURATION", time.Second); e != nil || d != 2*time.Second {
		t.Fatal(d, e)
	}
	t.Setenv("TEST_STRING", "")
	if _, e := String("TEST_STRING", "default"); e == nil {
		t.Fatal("empty accepted")
	}
}
