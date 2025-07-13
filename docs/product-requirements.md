# Product Requirements Document (PRD) – v1.6

**Project Name:** `itsjustintv`  
**Repository:** [rmoriz/itsjustintv](https://github.com/rmoriz/itsjustintv)  
**Version:** 1.6  
**Author:** Moriz GmbH  
**Date:** July 13, 2025  
**License:** MIT (c) 2025 Moriz GmbH

---

## 1. Objective

Build a configurable, self-hosted Go-based service that receives Twitch EventSub HTTP webhooks and notifies via downstream webhooks or file output when specified streamers go live. The system handles retries, deduplication, and includes enriched payloads with streamer metadata.

---

## 2. Functional Requirements

### 2.1 Streamer Monitoring (via Webhooks)

- Accept `stream.online` EventSub webhooks from Twitch at endpoint: **`/twitch`**
- Validate HMAC signature for authenticity
- Match event data to configured streamers (by login or user ID)
- Unwanted or stale subscriptions:
  - Respond with **HTTP `410 Gone`** to trigger immediate removal by Twitch

### 2.2 Webhook Dispatching

- Send JSON POST requests to:
  - A global default webhook (if enabled)
  - Or a streamer-specific override
- Optionally sign requests with HMAC
- Include streamer metadata from config and dynamic enrichment
- Webhook is only dispatched if:
  - Streamer matches
  - AND:
    - No `tag_filter` is configured
    - OR at least one **Twitch-provided tag** matches `tag_filter` (case-insensitive exact match)
- The final payload will include all tags (static + Twitch), regardless of filter

### 2.3 Web Server & TLS

- HTTP(S) server using Go's standard library
- **Optional** HTTPS via Let's Encrypt (`autocert`) - can be disabled for reverse proxy deployments
- Configurable listen address and optional ACME domains
- When ACME enabled:
  - Should renew in the background, store key+certs to disk (persistence)
  - Should check key/cert on start: Valid? => schedule renewal based on config, invalid? start provisioning immediately
  - Make sure TLS is working before starting with event-subscription/requests to Twitch
- When ACME disabled:
  - Run HTTP-only server (reverse proxy handles TLS)
  - Skip TLS validation checks
  - Proceed directly to Twitch subscription setup

### 2.4 Caching and Deduplication

- In-memory and optional persistent deduplication (file based, json, configurable location)
- Default TTL: 2h (configurable)

### 2.5 Retry Logic

- Exponential backoff retries for failed webhook deliveries:
  - Initial: 30s
  - Max window: 30min
- Retry state persisted in a JSON file

### 2.6 Output File (`output.json`)

- JSON file to store the last N webhook payloads (FIFO)
- Default: 25 entries
- Separate from retry file

### 2.7 Profile Image Caching

- Twitch profile image is fetched and cached per streamer for 7 days
- Cached image is embedded in payload as:

```json
"image": {
  "data": "<base64>",
  "mimeType": "image/jpeg"
}
```

### 2.8 Channel Metadata Enrichment

- On each `stream.online` event, enrich the payload with:
  - `url`: `https://twitch.tv/<login>`
  - `view_count`
  - `followers_count`
  - `tags`: merged set of:
    - `additional_tags` from config
    - dynamic tags from Twitch (API)
  - `language`:
    - `"german"` if Twitch tags contain "Deutsch"
    - `"english"` if Twitch tags contain "English"
    - default: `"english"`
  - `image`: see 2.7

- Errors in enrichment do not block webhook dispatch, should be logged.

### 2.9 Configuration

- Provided via `config.toml` and/or environment variables
- Configurable items include:
  - Twitch credentials
  - Server settings (port, HTTPS domains, cert path, cert renewal)
  - Retry, output, and telemetry settings
  - Global and per-streamer webhook targets

#### Example streamer config:

```toml
[streamers.twitch."streamer1"]
login = "streamer1"
description = "Tech and gamedev"
additional_tags = ["selfhosted", "german"]
tag_filter = ["Science & Technology", "Software Development"]
webhook_url = "https://example.com/webhook"
```

- `additional_tags`: static tags added to outgoing payload
- `tag_filter`: optional filter against Twitch-provided tags (not merged tags)

### 2.10 Twitch App Access Token Management

- Uses `client_credentials` flow to obtain app access token
- Automatically refreshed before expiration
- Stored in memory and on disc (persistence, json, including expiration date (absolute)). If persistence exists, key should be loaded and verified. If valid? use and schedule renewal based on the expiration date. If not: create new.

### 2.11 Twitch EventSub Subscription Refresh

- On startup:
  - Fetch current Twitch subscriptions, print to stdout status
  - Register missing ones based on config, print to stdout status
- Background task every 1h (+ splay 15min):
  - Validate subscription state
  - Re-register expired/missing subscriptions
- If a webhook is received for a no-longer-wanted streamer:
  - Respond with `410 Gone` to remove the subscription

### 2.12 Configuration Hot-Reloading

- Watch `config.toml` for changes (via `fsnotify`)
- On valid change:
  - Apply new config live
  - Add/remove/update subscriptions
- On failure: log error, retain previous config

### 2.13 CLI Interface & Flags

```bash
itsjustintv --help
itsjustintv --version
itsjustintv --config ./custom.toml
```

- `--help`: prints usage info
- `--version`: prints version and Git commit (if available)
- `--config`: path to config file (default: `config.toml`)

---

## 3. Non-Functional Requirements

- Performance: <1s response, 100+ streamers
- Reliability: retry, deduplication, reconnect
- Security: HMAC, TLS, no credential logging
- Persistence: retry state, image cache, and output file survive restarts

---

## 4. Observability

- OpenTelemetry support
- Export spans and metrics via OTLP (gRPC or HTTP)
- Metrics include delivery success/failure, retry attempts

---

## 5. Release Management

### 5.1 CI (GitHub Actions)

- Run `go test` and linter on each push to `main`
- Build and push Alpine-based Docker image to `ghcr.io/rmoriz/itsjustintv:latest`

### 5.2 Tagged Releases

- On Git tag (e.g. `v1.2.3`):
  - Static binary for `linux-amd64` (musl)
  - Docker image with tags: `v1.2.3`, `latest`
  - Create GitHub release

---

## 6. Documentation

- `README.md`: usage, Twitch OAuth, Docker
- `docs/product-requirements.md`: this PRD
- `docs/architecture.md`: system and flow overview
- `examples/payloads/`: real webhook payloads for testing

---

## 7. Milestones

| #   | Milestone                                 | Status | Target Date |
| --- | ----------------------------------------- | ------ | ----------- |
| 1   | Project scaffold + config loader          | ❌     | T+2 days    |
| 2   | HTTP server with autocert HTTPS           | ❌     | T+4 days    |
| 3   | Webhook receipt & validation              | ❌     | T+6 days    |
| 4   | Webhook dispatch logic w/ retry queue     | ❌     | T+8 days    |
| 5   | Metadata enrichment                       | ❌     | T+10 days   |
| 6   | Output + retry persistence (JSON files)   | ❌     | T+11 days   |
| 7   | OpenTelemetry integration                 | ❌     | T+12 days   |
| 8   | CI workflow, binary & image release       | ❌     | T+13 days   |
| 9   | Final QA, docs, README                    | ❌     | T+14 days   |

---

## 8. First Steps

1. Initialize GitHub repo using `gh`
2. add .gitignore for Go
3. Confirm this PRD
3. Draft `docs/architecture.md` before starting code


---

## 9. Git Discipline

- Descriptive, scoped commits
- AI-assisted commits must include:

```
AI-Generated-By: <Model Name> via <Agent Name>
```

- Tags for milestones (e.g. `milestone-2-https-server`)
- run tests, lint before every commmit
- commit every change
- push once a feature is complete



---

## 10. Extensibility for Other Platforms

Pluggable architecture via interface:

```go
type StreamEventProvider interface {
    Listen(handler func(StreamEvent)) error
}
```

Future providers (e.g. YouTube, Kick) can reuse infrastructure (caching, dispatching, etc.).
