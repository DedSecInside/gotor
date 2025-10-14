package crawl

import (
	"net/url"
	"time"
)

type Task struct {
	URL   *url.URL
	Depth int
	// Optional hint fields for metrics/trace could go here later.
}

type Options struct {
	Workers     int           // N workers
	QueueSize   int           // bounded frontier size
	Rate        float64       // RPS
	Burst       int           // token bucket burst
	MaxDepth    int           // inclusive max depth
	ReqTimeout  time.Duration // per-request timeout
	DialTimeout time.Duration // dial timeout (plumbed to netx)
}
