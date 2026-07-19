package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/DedSecInside/gotor/api"
	"github.com/DedSecInside/gotor/internal/crawl"
)

var version = "3.0.0-dev"

type config struct {
	url              string
	depth            int
	workers          int
	queue            int
	rps              float64
	burst            int
	outputFormat     string
	socksHost        string
	socksPort        int
	disableTor       bool
	perHostParallel  int
	perHostDelayMS   int
	maxResponseBytes int64
	allowNonHTML     bool
	randomizeHeaders bool
	serve            bool
	serverHost       string
	serverPort       int
	logLevel         string
	logFormat        string
	showVersion      bool
}

func parseFlags() config {
	var value config
	flag.StringVar(&value.url, "url", "", "URL to crawl")
	flag.IntVar(&value.depth, "depth", 1, "maximum crawl depth (0-10, root is depth 0)")
	flag.IntVar(&value.workers, "workers", 16, "number of crawler workers")
	flag.IntVar(&value.queue, "queue", 2048, "frontier queue size")
	flag.Float64Var(&value.rps, "rps", 5, "global requests per second")
	flag.IntVar(&value.burst, "burst", 5, "global rate-limiter burst")
	flag.StringVar(&value.outputFormat, "f", "list", "output format: list, tree, or json")
	flag.StringVar(&value.socksHost, "socks5-host", "127.0.0.1", "SOCKS5 proxy host")
	flag.IntVar(&value.socksPort, "socks5-port", 9050, "SOCKS5 proxy port")
	flag.BoolVar(&value.disableTor, "disable-socks5", false, "crawl without the Tor SOCKS5 proxy")
	flag.IntVar(&value.perHostParallel, "per-host-parallel", 2, "maximum concurrent requests per host")
	flag.IntVar(&value.perHostDelayMS, "per-host-delay", 0, "minimum delay between requests to one host (milliseconds)")
	flag.Int64Var(&value.maxResponseBytes, "max-response-bytes", 5*1024*1024, "largest response body to parse")
	flag.BoolVar(&value.allowNonHTML, "allow-non-html", false, "attempt to parse non-HTML responses")
	flag.BoolVar(&value.randomizeHeaders, "randomize-headers", false, "rotate browser user-agent headers")
	flag.BoolVar(&value.serve, "s", false, "run the job-control HTTP API")
	flag.StringVar(&value.serverHost, "server-host", "127.0.0.1", "HTTP API host")
	flag.IntVar(&value.serverPort, "server-port", 8081, "HTTP API port (0 chooses an available port)")
	flag.StringVar(&value.logLevel, "log-level", "info", "log level: debug, info, warn, or error")
	flag.StringVar(&value.logFormat, "log-format", "text", "log format: text or json")
	flag.BoolVar(&value.showVersion, "version", false, "print the GoTor version")
	flag.Parse()
	return value
}

func configureLogging(levelName, format string) error {
	var level slog.Level
	switch strings.ToLower(levelName) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return fmt.Errorf("unsupported log level %q", levelName)
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, options)
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, options)
	default:
		return fmt.Errorf("unsupported log format %q", format)
	}
	slog.SetDefault(slog.New(handler))
	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("gotor failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	value := parseFlags()
	if err := configureLogging(value.logLevel, value.logFormat); err != nil {
		return err
	}
	if value.showVersion {
		fmt.Println(version)
		return nil
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if value.serve {
		return api.NewServer(value.serverHost, value.serverPort).Run(ctx)
	}
	if value.url == "" {
		flag.Usage()
		return errors.New("-url is required unless -s is used")
	}

	request := crawl.Request{
		URL:               value.url,
		Depth:             value.depth,
		UseTor:            !value.disableTor,
		SOCKS5Host:        value.socksHost,
		SOCKS5Port:        value.socksPort,
		Workers:           value.workers,
		QueueSize:         value.queue,
		RequestsPerSecond: value.rps,
		Burst:             value.burst,
		PerHostParallel:   value.perHostParallel,
		PerHostDelayMS:    value.perHostDelayMS,
		MaxResponseBytes:  value.maxResponseBytes,
		AllowNonHTML:      value.allowNonHTML,
		RandomizeHeaders:  value.randomizeHeaders,
	}
	report, err := crawl.Execute(ctx, request)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	switch strings.ToLower(value.outputFormat) {
	case "json":
		return writeJSON(os.Stdout, report)
	case "list":
		writeList(os.Stdout, report)
	case "tree":
		writeTree(os.Stdout, report)
	default:
		return fmt.Errorf("unsupported output format %q", value.outputFormat)
	}
	return nil
}

func writeJSON(writer io.Writer, report crawl.Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeList(writer io.Writer, report crawl.Report) {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "DEPTH\tSTATUS\tTITLE\tURL")
	for _, page := range report.Pages {
		fmt.Fprintf(
			table,
			"%d\t%d\t%s\t%s\n",
			page.Depth,
			page.Metadata.Status,
			page.Metadata.Title,
			page.Metadata.URL,
		)
	}
	_ = table.Flush()
}

func writeTree(writer io.Writer, report crawl.Report) {
	pages := append([]crawl.PageResult(nil), report.Pages...)
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Depth == pages[j].Depth {
			return pages[i].Metadata.URL < pages[j].Metadata.URL
		}
		return pages[i].Depth < pages[j].Depth
	})
	for _, page := range pages {
		title := page.Metadata.Title
		if title == "" {
			title = page.Metadata.URL
		}
		fmt.Fprintf(writer, "%s%s (%d)\n", strings.Repeat("  ", page.Depth), title, page.Metadata.Status)
	}
}
