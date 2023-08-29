# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
