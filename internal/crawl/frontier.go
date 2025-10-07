// internal/crawl/frontier.go
package crawl

import (
	"sync"
)

type Frontier struct {
	ch       chan Task
	mu       sync.Mutex
	seen     map[string]struct{}
	inflight int
	closed   bool
	once     sync.Once
}

func NewFrontier(size int) *Frontier {
	return &Frontier{
		ch:   make(chan Task, size),
		seen: make(map[string]struct{}),
	}
}

func (f *Frontier) EnqueueIfNew(t Task) bool {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return false
	}
	key := t.URL.String()
	if _, ok := f.seen[key]; ok {
		f.mu.Unlock()
		return false
	}
	f.seen[key] = struct{}{}
	f.mu.Unlock()

	select {
	case f.ch <- t:
		return true
	default:
		// bounded queue backpressure: drop or switch to blocking send if preferred
		return false
	}
}

func (f *Frontier) Next() <-chan Task { return f.ch }

func (f *Frontier) markStart() {
	f.mu.Lock()
	f.inflight++
	f.mu.Unlock()
}

func (f *Frontier) markDone() {
	f.mu.Lock()
	f.inflight--
	f.mu.Unlock()
}

func (f *Frontier) Stats() (queued, inflight int, closed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.ch), f.inflight, f.closed
}

func (f *Frontier) CloseOnce() {
	f.once.Do(func() {
		f.mu.Lock()
		f.closed = true
		close(f.ch)
		f.mu.Unlock()
	})
}
