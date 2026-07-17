# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

- Versioned JSON crawl reports for CLI and application integrations.
- REST job control with start, inspect, cancel, health, and readiness endpoints
  (#72).
- Tor configuration diagnostics with SOCKS reachability, `torrc` discovery,
  configured ports, data directory, and executable discovery.
- Parent relationships, page metadata, email addresses, phone numbers, and
  partial crawl errors in reports.
- Per-host parallelism and delay controls (#74).
- Optional randomized browser user-agent headers (#44).
- Structured text or JSON logging with configurable levels (#76).
- Response-size and content-type guardrails, `<base>` URL resolution, and meta
  refresh discovery (#77).
- Race-tested coverage for crawler relationships, contact extraction,
  politeness limits, job cancellation, and Tor diagnostics.

### Changed

- Replaced crawler log output with deterministic list, tree, and JSON formats.
- Hardened the container as a non-root multi-stage distroless image.
- Updated CI to enforce formatting, vetting, race tests, coverage, and builds.
- Reworked the README and API reference around the current v3 crawler.

### Fixed

- Prevented the frontier from closing while a worker is still processing its
  final queued page.
- Restored the HTTP service removed during the concurrency rewrite.
- Corrected the stale Docker build path.

## v2.0.0 - 2023-08-28
Bug fixes, slight performance updates, and some refactoring

## What's Changed
* Add logging by @KingAkeem in https://github.com/DedSecInside/gotor/pull/35
* Add configuration package and .env file, utilize env variables by @KingAkeem in https://github.com/DedSecInside/gotor/pull/37
* Add docker by @KingAkeem in https://github.com/DedSecInside/gotor/pull/39
* Changed `USE_TOR` default value to `true`. by @fukusuket in https://github.com/DedSecInside/gotor/pull/40
* Refactoring and improving documentation by @KingAkeem in https://github.com/DedSecInside/gotor/pull/41
* Improving docker by @KingAkeem in https://github.com/DedSecInside/gotor/pull/42
* Bump golang.org/x/net from 0.4.0 to 0.7.0 by @dependabot in https://github.com/DedSecInside/gotor/pull/47
* Update README.md by @KingAkeem in https://github.com/DedSecInside/gotor/pull/48

## New Contributors
* @fukusuket made their first contribution in https://github.com/DedSecInside/gotor/pull/40
* @dependabot made their first contribution in https://github.com/DedSecInside/gotor/pull/47

**Full Changelog**: https://github.com/DedSecInside/gotor/compare/v1.0.1...v2.0.0

## v1.0.1 - 2022-11-10

### What's Changed
* Feature #22: Crawl phone numbers by @PSNAppz in https://github.com/DedSecInside/gotor/pull/25
* Add struct tags by @KingAkeem in https://github.com/DedSecInside/gotor/pull/26
* Patch: Get phone number if exists by @PSNAppz in https://github.com/DedSecInside/gotor/pull/27
* Get Web Content API by @PSNAppz in https://github.com/DedSecInside/gotor/pull/28
* Reorganize code and update documentation by @KingAkeem in https://github.com/DedSecInside/gotor/pull/29

### New Contributors
* @PSNAppz made their first contribution in https://github.com/DedSecInside/gotor/pull/25

**Full Changelog**: https://github.com/DedSecInside/gotor/compare/v1.0.0...v1.0.1

## v1.0.0 - 2021-09-28

### What's in this release?

- Resolves issues with depth argument when crawling a node/building a tree
- Unit tests for tree logic
- Create token stream pipeline
