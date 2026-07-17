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
	pending  int
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
	defer f.mu.Unlock()
	if f.closed {
		return false
	}
	key := t.URL.String()
	if _, ok := f.seen[key]; ok {
		return false
	}

	select {
	case f.ch <- t:
		f.seen[key] = struct{}{}
		f.pending++
		return true
	default:
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
	f.pending--
	f.mu.Unlock()
}

func (f *Frontier) Stats() (queued, inflight, pending int, closed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.ch), f.inflight, f.pending, f.closed
}

func (f *Frontier) CloseOnce() {
	f.once.Do(func() {
		f.mu.Lock()
		f.closed = true
		close(f.ch)
		f.mu.Unlock()
	})
}
