package calculator

import (
	"example.com/nativecalculator/internal/metrics"
	"example.com/nativecalculator/internal/native"
	"sync"
	"time"
)

type shard struct {
	mu       sync.RWMutex
	sum, sub int64
}
type Processor struct {
	queue  chan int64
	shards []shard
	wg     sync.WaitGroup
}

func NewProcessor(workers, queueSize int, lib native.Calculator, m *metrics.Metrics) *Processor {
	p := &Processor{queue: make(chan int64, queueSize), shards: make([]shard, workers)}
	for i := range p.shards {
		p.wg.Add(1)
		go func(s *shard) {
			defer p.wg.Done()
			var sum, sub int64
			for num := range p.queue {
				started := time.Now()
				sum = lib.Add(sum, num)
				m.C.Observe(time.Since(started).Seconds())
				started = time.Now()
				sub = lib.Sub(sub, num)
				m.Rust.Observe(time.Since(started).Seconds())
				s.mu.Lock()
				s.sum, s.sub = sum, sub
				s.mu.Unlock()
			}
		}(&p.shards[i])
	}
	return p
}

// Drain is called once, only after all HTTP handlers have left the admission gate.
func (p *Processor) Drain() { close(p.queue); p.wg.Wait() }
func (p *Processor) Snapshot() (sum, sub int64) {
	for i := range p.shards {
		s := &p.shards[i]
		s.mu.RLock()
		sum += s.sum
		sub += s.sub
		s.mu.RUnlock()
	}
	return
}
