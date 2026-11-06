---
feature_code: FEAT-003
service: sandbox-svc
spec_version: 1
last_updated: 2026-11-05
---

# FEAT-003 — App Flow

<!-- BEGIN:DOCPREP -->
## Overview

Backend-only feature (`has_ui: false`). An authenticated user requests a
CSV export of their own favorites and receives a file; there is no screen
or page in this feature.

## Key User Journeys

- **Export favorites (REQ-001, REQ-002, REQ-003, REQ-004):** Authenticated
  user calls `GET /favorites/export` → receives a `text/csv` attachment
  containing a header row plus one row per current favorite,
  newest-favorited first.
- **Export an empty list (REQ-011):** User with zero favorites calls the
  same endpoint → receives a `200` CSV containing only the header row.
- **Anonymous attempt (REQ-012):** Caller without a valid session calls
  the endpoint → receives the standard `401 UNAUTHENTICATED` envelope and
  no file.
- **Cross-user attempt (REQ-013):** Caller tries to influence whose
  favorites are exported → impossible, because the user id comes only
  from the auth context; there is no user parameter and no admin path.
- **Discontinued favorite (REQ-014):** A favorited item has since been
  marked unavailable → it still appears in the export with `available` =
  `Tidak Tersedia`.

## Main Pages / Actions

- No pages — backend-only, `has_ui: false`.
- Action: `GET /favorites/export` → CSV download.
<!-- END:DOCPREP -->

<!-- BEGIN:CODEPREP -->
## Routes

| Method | Path | Consumer | Purpose |
|---|---|---|---|
| `GET` | `/v1/favorites/export` | `external` | Stream the calling user's own favorites as a `text/csv` attachment. No parameters, no body. |

No frontend routes — `has_ui: false`.

## Components

- **`favorite_endpoint` (raw handler)** — `NewExportHandler`, registered
  on the service's `mux` router rather than through `WrapService`,
  because nexcommon's response writer cannot emit `text/csv` with a
  quoted `Content-Disposition`. Authenticates with the same
  `UserAccessValidator()` the wrapped routes use.
- **`favorite.Service.ExportCSV`** — reads the context user id, calls
  the DAO, renders the file; the pure `buildCSV` helper owns the BOM,
  header, CRLF terminator, RFC 4180 quoting, availability labels, UTC
  date and formula neutralization.
- **`favorite_dao.ListForExport`** — one query over
  `favorite_items LEFT JOIN catalog_items`, ordered
  `created_at DESC, item_id ASC`, mirroring FEAT-001's `List`.
- **Migration `3_export_columns.sql`** — adds `catalog_items.category`,
  `catalog_items.price` and `favorite_items.note`; no new table, no
  index, no backfill.
- **Contract artifacts** — `api/openapi.yaml` and `postman/` are
  hand-authored for this route; nexcommon's generator only sees wrapped
  routes.
<!-- END:CODEPREP -->
