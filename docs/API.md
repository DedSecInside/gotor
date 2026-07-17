# GoTor HTTP API

The API is versioned under `/api/v1`. Requests and responses use JSON.

## Health

`GET /_healthz` reports that the process is alive.

`GET /_readyz` reports that the HTTP service is ready to accept jobs.

## Inspect Tor configuration

`GET /api/v1/tor/status?host=127.0.0.1&port=9050`

GoTor checks whether the selected SOCKS endpoint accepts TCP connections and
searches standard platform-specific `torrc` locations. `TORRC_PATH` takes
precedence when set.

```json
{
  "configured": true,
  "reachable": true,
  "host": "127.0.0.1",
  "port": 9050,
  "configPath": "/home/user/.config/tor/torrc",
  "socksPorts": ["127.0.0.1:9050"],
  "controlPorts": ["9051"],
  "dataDirectory": "/home/user/.local/share/tor",
  "torBinary": "/usr/bin/tor",
  "portMatches": true,
  "searchedPaths": [
    "/etc/tor/torrc",
    "/home/user/.config/tor/torrc"
  ],
  "checkedAt": "2026-07-17T12:00:00Z"
}
```

`configured` and `reachable` are intentionally separate. A remotely managed
Tor proxy may be reachable without a local `torrc`; a local config may also be
present while the Tor process is stopped. `portMatches` is omitted when the
configuration uses `auto` or a Unix socket.

## Start a job

`POST /api/v1/jobs`

```json
{
  "url": "https://example.com",
  "depth": 1,
  "useTor": true,
  "socks5Host": "127.0.0.1",
  "socks5Port": 9050,
  "workers": 16,
  "queueSize": 2048,
  "requestsPerSecond": 5,
  "burst": 5,
  "perHostParallel": 2,
  "perHostDelayMs": 0,
  "maxResponseBytes": 5242880,
  "allowNonHtml": false,
  "randomizeHeaders": true
}
```

Only `url`, `depth`, and `useTor` normally need to be supplied. Proxy and
crawler controls use safe defaults. A successful request returns `202
Accepted` and a job snapshot.

## Inspect a job

`GET /api/v1/jobs/{id}`

Statuses are `queued`, `running`, `completed`, `failed`, and `cancelled`.
Completed and cancelled jobs include the available crawl report:

```json
{
  "id": "4e6a...",
  "status": "completed",
  "report": {
    "schemaVersion": 1,
    "engine": "gotor",
    "target": "https://example.com",
    "maxDepth": 1,
    "usesTor": false,
    "startedAt": "2026-07-17T12:00:00Z",
    "durationMs": 245,
    "pages": [
      {
        "depth": 0,
        "links": ["https://www.iana.org/help/example-domains"],
        "metadata": {
          "url": "https://example.com",
          "title": "Example Domain",
          "status": 200,
          "contentType": "text/html"
        }
      }
    ],
    "errors": []
  }
}
```

Each page includes its `parentUrl`, crawl depth, discovered links, metadata,
email addresses, and phone numbers when available. Skipped binary or oversized
content includes `skippedReason`. Fetch failures appear in `report.errors`.

## Cancel a job

`POST /api/v1/jobs/{id}/cancel`

Cancellation propagates to in-flight HTTP requests. The terminal job snapshot
retains any pages collected before cancellation.

## Error responses

Errors use the following shape:

```json
{"error":"depth must be between 0 and 10"}
```

Invalid input returns `400`; unknown job IDs return `404`.
