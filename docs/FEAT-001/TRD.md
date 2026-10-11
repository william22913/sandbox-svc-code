---
feature_code: FEAT-001
service: sandbox-svc
locked_version: 1
last_updated: 2026-10-10
---

# FEAT-001 — Technical Requirements Document (backend half)

> Overview summary only — see `architecture.md` for the full design and
> its rationale.

## Overview

Favorite Items adds a personal, per-user saved-items list to sandbox-svc:
favorite/un-favorite an item from its detail page, and a "My Favorites"
page listing them, with discontinued items shown as "unavailable" rather
than vanishing. Backend-only this pass — no frontend scope yet.

## Technical Goals

- Stand up `sandbox-svc-code` (currently an empty greenfield repo) with
  its first real feature, on the team's standard Go + nexcommon stack.
- Make favoriting idempotent and capped (max 500 per user) without ever
  silently failing or evicting an existing favorite.
- Satisfy REQ-004 (discontinued items stay listed, marked unavailable)
  via a live read-time join rather than denormalized data — confirmed
  safe now that catalog_items rows are known to never be hard-deleted.

## Technology Stack

| Area | Tool | Purpose |
|---|---|---|
| Language/runtime | Go | Team-standard backend language for this pipeline. |
| Shared library | `nexcommon` | Controller wrapping, DTO/validation, DAO/repository pattern, request-context user ID, Prometheus metrics middleware. |
| Persistence | Postgres | Single new table `favorite_items`; no separate datastore needed for this simple relation. |
| Auth | Assumed shared middleware (consumed, not built here) | Places authenticated user ID into request context; gates anonymous access (REQ-006) via 401 before handlers run. |
| Observability | nexcommon Prometheus middleware | Standard inbound HTTP request metrics (method/route/status/latency) wired once for this greenfield service. |
