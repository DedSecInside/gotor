package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DedSecInside/gotor/internal/crawl"
)

func main() {
	// existing flags…
	urlFlag := flag.String("url", "", "URL to crawl")
	depth := flag.Int("depth", 1, "Depth of the tree (inclusive)")

	// new but optional knobs with safe defaults
	workers := flag.Int("workers", 16, "Number of crawler workers")
	queue := flag.Int("queue", 2048, "Frontier queue size")
	rps := flag.Float64("rps", 5.0, "Global requests per second")
	burst := flag.Int("burst", 5, "Limiter burst")
	outFmt := flag.String("f", "list", "Output format: list or tree")

	// Tor opts you already expose elsewhere (reusing your flags/vars)
	socksHost := flag.String("socks5-host", "127.0.0.1", "SOCKS5 host")
	socksPort := flag.Int("socks5-port", 9050, "SOCKS5 port")
	disableTor := flag.Bool("disable-socks5", false, "Disable SOCKS5/Tor")

	flag.Parse()

	// If running server mode, fall through to existing API startup path (unchanged).
	// if *serverMode { startServer(); return }

	if *urlFlag == "" {
		flag.Usage()
		return
	}

	log.Printf("crawl start: url=%s workers=%d queue=%d rps=%.2f burst=%d", *urlFlag, *workers, *queue, *rps, *burst)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SIGINT/SIGTERM → graceful cancel
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("signal received, shutting down…")
		cancel()
	}()

	c, err := crawl.NewCrawler(ctx, crawl.Options{
		Workers:     *workers,
		QueueSize:   *queue,
		Rate:        *rps,
		Burst:       *burst,
		MaxDepth:    *depth,
		ReqTimeout:  30 * time.Second,
		DialTimeout: 10 * time.Second,
	}, !*disableTor, *socksHost, *socksPort)
	if err != nil {
		log.Fatalf("crawler init error: %v", err)
	}

	if c.Seed(*urlFlag) == 0 {
		log.Fatalf("invalid seed url: %s", *urlFlag)
	}

	if err := c.Run(ctx, *outFmt); err != nil && err != context.Canceled {
		log.Fatalf("crawl error: %v", err)
	}
}
