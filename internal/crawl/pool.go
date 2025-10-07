package crawl

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/DedSecInside/gotor/internal/netx"
)

type Crawler struct {
	opts     Options
	frontier *Frontier
	client   *http.Client
	limiter  *rate.Limiter
}

func NewCrawler(ctx context.Context, opts Options, useTor bool, socksHost string, socksPort int) (*Crawler, error) {
	httpClient, err := netx.NewHTTPClient(netx.ClientOpts{
		UseTor:       useTor,
		SocksHost:    socksHost,
		SocksPort:    socksPort,
		ReqTimeout:   opts.ReqTimeout,
		DialTimeout:  opts.DialTimeout,
		MaxIdleConns: 32,
	})
	if err != nil {
		return nil, err
	}
	return &Crawler{
		opts:     opts,
		frontier: NewFrontier(opts.QueueSize),
		client:   httpClient,
		limiter:  rate.NewLimiter(rate.Limit(opts.Rate), opts.Burst),
	}, nil
}

// Seed one or more starting URLs at depth 0.
func (c *Crawler) Seed(rawURLs ...string) int {
	added := 0
	for _, ru := range rawURLs {
		u, err := url.Parse(ru)
		if err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		if c.frontier.EnqueueIfNew(Task{URL: u, Depth: 0}) {
			added++
		}
	}
	return added
}

func (c *Crawler) Run(ctx context.Context, outFormat string) error {
	g, ctx := errgroup.WithContext(ctx)

	// Track active workers for drain-aware shutdown.
	c.frontier.incWorkers(c.opts.Workers)

	for i := 0; i < c.opts.Workers; i++ {
		g.Go(func() error {
			defer c.frontier.decWorkers()

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case t, ok := <-c.frontier.Next():
					if !ok {
						return nil // frontier closed
					}

					// Rate-limit globally.
					if err := c.limiter.Wait(ctx); err != nil {
						return err
					}

					if err := c.process(ctx, t, outFormat); err != nil {
						// Log and continue; worker should not die unless context cancelled.
						log.Printf("process error: %v", err)
					}
				}
			}
		})
	}

	// Wait until frontier drains to a terminal state for MaxDepth, then close.
	// We don’t spin: we poll light-weight with a timer to avoid busy-waiting.
	t := time.NewTicker(150 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			c.frontier.Close()
			_ = g.Wait()
			return ctx.Err()
		case <-t.C:
			// Terminate condition: no more enqueues possible (all nodes at MaxDepth already expanded)
			// AND queue drained AND workers idle. We infer “no more enqueues” because process()
			// never enqueues beyond MaxDepth.
			if c.frontier.DrainAwareDone() && len(c.frontier.Next()) == 0 {
				c.frontier.Close()
				return g.Wait()
			}
		}
	}
}

// extractLinks parses <a href> tags from the body, resolves them against base,
// and returns absolute URLs suitable for enqueueing.
func extractLinks(r io.Reader, base *url.URL) ([]*url.URL, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var links []*url.URL

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					href := strings.TrimSpace(a.Val)
					if href == "" || strings.HasPrefix(href, "#") {
						continue
					}
					// Resolve relative paths and filter junk
					u, err := base.Parse(href)
					if err != nil || u.Scheme == "mailto" || u.Scheme == "javascript" {
						continue
					}
					u.Fragment = "" // drop fragments
					if _, ok := seen[u.String()]; !ok {
						seen[u.String()] = struct{}{}
						links = append(links, u)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return links, nil
}

func (c *Crawler) process(ctx context.Context, t Task, outFormat string) error {
	// Build request with netx (keeps Tor + timeouts aligned with project).
	req, err := netx.NewRequest(ctx, http.MethodGet, t.URL.String(), "GoTor/1.0")
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	links, err := extractLinks(resp.Body, t.URL)
	if err != nil {
		log.Printf("failed to parse links from %s: %v", t.URL, err)
	}

	// Emit/record node according to outFormat (list/tree). Keep today’s behavior:
	switch strings.ToLower(outFormat) {
	case "tree":
		// no-op here; assume caller aggregates via linktree package
	default:
		// list: log line for now (existing API/handlers can keep their own sinks)
		log.Printf("depth=%d url=%s status=%d links=%d", t.Depth, t.URL, resp.StatusCode, len(links))
	}

	// Enqueue children if we’re not at max depth yet.
	nextDepth := t.Depth + 1
	if nextDepth <= c.opts.MaxDepth {
		for _, child := range links {
			_ = c.frontier.EnqueueIfNew(Task{URL: child, Depth: nextDepth})
		}
	}
	return nil
}
