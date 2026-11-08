---
feature_code: FEAT-005
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — Technical Requirements Document

> Overview summary only — see `architecture.md` for the full design and
> its rationale.

## Overview

Favorites summary and badge tooltip adds one read-only endpoint and one
hover overlay on an existing badge. `GET /v1/favorites/summary` returns
the calling user's own `total` favorites and how many of them reference a
currently discontinued catalog item, and FEAT-004's header badge shows
`N favorites (M discontinued)` in a tooltip on first hover. No schema
change, no new index, no cache, no counter column, and no change to any
existing endpoint.

## Technical Goals

- Answer both counts from one aggregate query over FEAT-001's
  `favorite_items` joined live to `catalog_items`:
  `COUNT(*)` for `total` and
  `COUNT(*) FILTER (WHERE NOT COALESCE(c.is_available, FALSE))` for
  `discontinued`.
- Keep `total` inclusive of discontinued favorites and of favorites whose
  catalog row is missing; the `FILTER` only narrows `discontinued`, so
  `0 <= discontinued <= total` by construction.
- Scope the query only by the auth-context user id. The endpoint declares
  no query, path or body parameters, so no caller input can select
  another user's rows.
- Answer `200` with `payload.total` and `payload.discontinued` for zero
  favorites — a normal `{0, 0}`, never `404` or an empty body.
- Keep the read cheap and stateless: the per-user row set is capped at
  500, `user_id` leads the existing composite PK, and there is no
  transaction, lock or write.
- Expose the route through nexcommon's standard wrapped endpoint, so it
  is authenticated by the shared `UserAccessValidator()`, observed by the
  existing status-aware metrics middleware, and visible to the swagger
  generator with no new wiring.
- In the UI, fetch the summary on the first hover of a visible badge only
  — never on app load — never refetch on repeat hover, retry on a later
  hover after a failure, and invalidate any held summary after a
  successful favorite/un-favorite so pre-action values can never be
  shown.
- Keep the tooltip display-only and hover-only: no click action, no
  spinner, no error UI, and no effect on the badge's own count, `99+`
  cap or visibility rules.
- Add no analytics, polling, cross-tab sync, per-item breakdown, admin
  view or new page — the spec's Non-Goals.

## Technology Stack

| Area | Tool | Purpose |
|---|---|---|
| Language/runtime | Go 1.27 | Unchanged from FEAT-001/FEAT-003/FEAT-004; no new dependency. |
| Shared library | `nexcommon v0.3.68-swagger` | `WrapService`, `UserAccessValidator`, error formatter and DTO helpers — reused, not re-implemented. |
| HTTP router | `gorilla/mux` | One new wrapped route on the existing router; static path, no ordering conflict with `GET /v1/favorites`. |
| Persistence | PostgreSQL (`database/sql` + pgx) | One aggregate `SELECT` with a `LEFT JOIN` to `catalog_items`; served by the leading column of the existing composite PK. |
| Migrations | `sql-migrate` | None for this feature — no schema change. |
| Auth | `nexcommon` `UserAccessValidator()` over Redis | Same validator as the existing favorite routes; missing token -> `401 E-1-CMD-AUT-001`. |
| Observability | Existing Prometheus stack | `/metrics` and `StatusAwareMetricsMiddleware` already installed; the new route is covered with no new collector, metric or dashboard. |
| Contract artifacts | `api/openapi.yaml`, `postman/` | The wrapped route is visible to nexcommon's swagger generator; `/go-dev` carries it into both. |
| Frontend | React 18 + Vite + Ant Design + `nexcomponent-ui` | Extends FEAT-004's `FavoriteCountProvider`/badge with summary state and an Ant Design `Tooltip`; no new dependency, no new route. |
