//go:build cgo && linux

package native

import (
	"math"
	"testing"
)

func TestFFI(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if c.Add(10, 5) != 15 || c.Sub(10, 5) != 5 {
		t.Fatal("incorrect native result")
	}
	if c.Add(math.MaxInt64, 1) != math.MinInt64 || c.Sub(math.MinInt64, 1) != math.MaxInt64 {
		t.Fatal("incorrect wraparound")
	}
}
