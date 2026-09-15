package metrics

import (
	"sync"
	"time"
)

// 61 slots preserve the oldest complete second while current-second writes occur.
type bucket struct {
	second int64
	count  uint64
}
type RPSWindow struct {
	mu    sync.Mutex
	slots [61]bucket
	now   func() time.Time
}

func NewRPSWindow(now func() time.Time) *RPSWindow { return &RPSWindow{now: now} }
func index(second int64) int                       { return int((second%61 + 61) % 61) }
func (w *RPSWindow) Increment() {
	w.mu.Lock()
	defer w.mu.Unlock()
	second := w.now().Unix()
	b := &w.slots[index(second)]
	if b.second != second {
		*b = bucket{second: second}
	}
	b.count++
}
func (w *RPSWindow) Snapshot() (result [60]uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now().Unix()
	for ago := int64(1); ago <= 60; ago++ {
		b := w.slots[index(now-ago)]
		if b.second == now-ago {
			result[ago-1] = b.count
		}
	}
	return
}
