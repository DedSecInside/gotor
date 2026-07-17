# GoTor

GoTor is a concurrent web crawler with first-class Tor SOCKS5 support. It can
run as a human-friendly CLI, emit a versioned JSON crawl report, or expose a
local job-control HTTP API for applications such as TorBot App.

## Requirements

- Go 1.24 or newer
- A Tor SOCKS5 proxy on `127.0.0.1:9050` by default (optional)

## Quick start

```bash
go build -o bin/gotor ./cmd/main

# Crawl through Tor
./bin/gotor -url https://example.com -depth 1

# Crawl directly and emit the integration-safe JSON schema
./bin/gotor \
  -url https://example.com \
  -depth 1 \
  -disable-socks5 \
  -f json
```

Depth is inclusive: the root is depth `0`, so `-depth 1` fetches the root and
its direct links.

## HTTP API

Start the service:

```bash
./bin/gotor -s -server-host 127.0.0.1 -server-port 8081
```

Start and inspect a crawl job:

```bash
curl -sS \
  -H 'content-type: application/json' \
  -d '{
    "url": "https://example.com",
    "depth": 1,
    "useTor": false,
    "randomizeHeaders": true
  }' \
  http://127.0.0.1:8081/api/v1/jobs

curl -sS http://127.0.0.1:8081/api/v1/jobs/JOB_ID
curl -sS -X POST http://127.0.0.1:8081/api/v1/jobs/JOB_ID/cancel
```

Health and readiness probes are available at `/_healthz` and `/_readyz`.
Tor configuration and proxy readiness are available at:

```bash
curl -sS 'http://127.0.0.1:8081/api/v1/tor/status?host=127.0.0.1&port=9050'
```

The diagnostic reports whether the selected SOCKS endpoint is reachable, the
detected `torrc` path, configured SOCKS and control ports, Tor data directory,
and Tor executable. Set `TORRC_PATH` when Tor uses a non-standard
configuration location.

See [docs/API.md](docs/API.md) for the full request and response contract.

## Crawl controls

```text
-url URL                  Root URL to crawl
-depth N                  Maximum depth, 0-10 (default 1)
-workers N                Global worker count (default 16)
-queue N                  Frontier capacity (default 2048)
-rps N                    Global requests per second (default 5)
-burst N                  Global limiter burst (default 5)
-per-host-parallel N      Concurrent requests to one host (default 2)
-per-host-delay MS        Delay between requests to one host
-max-response-bytes N     Largest response parsed in memory (default 5 MiB)
-randomize-headers        Rotate browser user-agent headers
-allow-non-html           Attempt to parse non-HTML responses
-socks5-host HOST         Tor SOCKS5 host (default 127.0.0.1)
-socks5-port PORT         Tor SOCKS5 port (default 9050)
-disable-socks5           Use a direct connection
-f list|tree|json         CLI output format
-log-format text|json     Structured log format
-log-level LEVEL          debug, info, warn, or error
```

GoTor skips oversized and non-HTML bodies by default but records those pages
and the skip reason in the report. Per-page failures are also returned without
discarding successful results.

## Container

The image is a multi-stage, non-root distroless build:

```bash
docker build -t gotor .
docker run --rm -p 8081:8081 gotor
```

If the container needs a Tor proxy, attach it to the relevant network and pass
`-socks5-host` when overriding the image command.

## Development

```bash
go fmt ./...
go vet ./...
go test -race ./...
```

GoTor is licensed under the GNU General Public License. See [LICENSE](LICENSE).
