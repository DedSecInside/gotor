package crawl

import (
	"context"
	"sync"
)

type Frontier struct {
	ch    chan Task
	seen  map[string]struct{}
	mu    sync.Mutex
	openW int // number of workers currently running
	wg    sync.WaitGroup
}

func NewFrontier(size int) *Frontier {
	return &Frontier{
		ch:   make(chan Task, size),
		seen: make(map[string]struct{}),
	}
}

func (f *Frontier) EnqueueIfNew(t Task) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := t.URL.String()
	if _, ok := f.seen[key]; ok {
		return false
	}
	f.seen[key] = struct{}{}
	select {
	case f.ch <- t:
		return true
	default:
		// queue full -> drop; upstream backpressure will naturally slow via limiter
		// (alternatively, block here if you prefer harder backpressure)
		return false
	}
}

func (f *Frontier) Next() <-chan Task { return f.ch }

func (f *Frontier) Close() { close(f.ch) }

func (f *Frontier) SeenCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}

// Helper used by pool to know when all workers are done.
func (f *Frontier) incWorkers(n int) { f.mu.Lock(); f.openW += n; f.mu.Unlock() }
func (f *Frontier) decWorkers()      { f.mu.Lock(); f.openW--; f.mu.Unlock() }
func (f *Frontier) workersAlive() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.openW
}

// DrainAwareDone tells if we can terminate: channel drained and no workers alive.
func (f *Frontier) DrainAwareDone() bool {
	return f.workersAlive() == 0 && len(f.ch) == 0
}

func (f *Frontier) Wait(ctx context.Context) {
	// For future expansion if we switch to internal waitgroup semantics
	<-ctx.Done()
}
