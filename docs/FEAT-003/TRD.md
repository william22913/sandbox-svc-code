---
feature_code: FEAT-003
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-06
---

# FEAT-003 — Technical Requirements Document

> Overview summary only — see `architecture.md` for the full design and
> its rationale.

## Overview

Export my favorites to CSV adds one read-only endpoint to
`sandbox-svc`: `GET /v1/favorites/export` returns the calling user's own
favorites as a downloadable `text/csv` file, newest-favorited first,
with a fixed seven-column header. Backend-only — `has_ui: false` — and
read-only: no existing endpoint, response shape or table definition
changes. Three columns the export needs are added to tables FEAT-001
already created.

## Technical Goals

- Serve the export without changing any existing route or response
  shape, reusing FEAT-001's `favorite_items` join to `catalog_items`.
- Produce a file that satisfies the format contract exactly: UTF-8 with
  BOM, CRLF records, RFC 4180 quoting with doubled quotes, comma
  delimiter, raw unformatted prices, `YYYY-MM-DD` UTC dates, and
  `Tersedia` / `Tidak Tersedia` labels.
- Neutralize CSV formula injection (`=`, `+`, `-`, `@` as the first
  character of `name`, `category` or `note`) without altering any other
  value.
- Scope every read to the authenticated user's id from the auth
  context — no parameter, header or body field can select another user.
- Return `200` with a header-only CSV for a user with zero favorites,
  and the standard `401` envelope for anonymous or invalid tokens.
- Keep the schema change additive and minimal: three columns, one
  batch, no new table, no index, no backfill.

## Technology Stack

| Area | Tool | Purpose |
|---|---|---|
| Language/runtime | Go 1.27 | Unchanged from FEAT-001; no new dependency. |
| Shared library | `nexcommon v0.3.68-swagger` | Route registration, auth validator, error formatter, DTO/converter helpers — reused, not re-implemented. |
| HTTP router | `gorilla/mux` | The export is a raw `mux` route on the same router, so it inherits the existing metrics middleware. |
| Persistence | PostgreSQL (`database/sql` + pgx) | One read query over `favorite_items LEFT JOIN catalog_items`; `trim_scale(price)::text` preserves exact decimal values. |
| Migrations | `sql-migrate` | One additive batch, `sql_migrations/main/3_export_columns.sql`. |
| CSV rendering | Hand-rolled RFC 4180 writer | `encoding/csv` with `UseCRLF` mutates embedded CR/LF inside quoted fields, which REQ-009 forbids. |
| Auth | `nexcommon` `UserAccessValidator()` over Redis | Same validator as the existing routes; missing token -> `401 E-1-CMD-AUT-001`. |
| Observability | Existing Prometheus stack | `/metrics` and `StatusAwareMetricsMiddleware` already installed; the new route is covered with no new collector or dashboard. |
| Contract artifacts | `api/openapi.yaml`, `postman/` | Hand-authored for this route — nexcommon's generator only sees wrapped routes. |
| Frontend | None | `has_ui: false`; the export is API/backend only. |
