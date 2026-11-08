---
feature_code: FEAT-003
service: sandbox-svc
spec_version: 1
last_updated: 2026-11-05
---

# FEAT-003 — Favorites CSV Export (PRD)

## Overview

Employees who maintain favorites currently have no way to move that list
into a spreadsheet; the only option is copying rows by hand from the
favorites list or API. This feature adds a backend-only export that
returns the caller's own favorites as a CSV file for monthly reporting.
See [spec.md](spec.md) for the full requirement detail.

## Product Goal

Let any authenticated user download their own favorites as a single CSV
that opens cleanly in a spreadsheet, without adding filters, other
formats, scheduling, or any UI in this pass.

## Features

| Feature | Description | Expected behavior |
|---|---|---|
| Trigger export | `GET /favorites/export` | Returns a `text/csv` attachment named `favorites-<user_id>-<YYYY-MM-DD>.csv` (UTC export date) (REQ-001, REQ-002) |
| Export contents | One row per current favorite | All of the user's favorites, newest-favorited first, with `item_id,name,category,price,available,favorited_at,note` (REQ-003, REQ-004) |
| Value formatting | How each column is written | Raw numeric price, ISO `YYYY-MM-DD` UTC date, `Tersedia`/`Tidak Tersedia`, verbatim note or empty cell (REQ-005, REQ-006, REQ-007, REQ-008) |
| Spreadsheet-safe file | Encoding, quoting, and formula neutralization | UTF-8 with BOM, CRLF line endings, RFC 4180 quoting, leading `=`, `+`, `-`, `@` neutralized in `name`/`category`/`note` (REQ-009, REQ-010) |
| Empty favorites | User with no favorites exports | `200` with a header-only CSV, not an error (REQ-011) |
| Access control | Anonymous or cross-user attempt | `401 UNAUTHENTICATED` for anonymous callers; export is always scoped to the auth-context user, and no parameter can select another user (REQ-012, REQ-013) |
| Discontinued items | A favorited item is no longer available | Still exported, marked `Tidak Tersedia` (REQ-014) |
| Live values | Item fields change before export | Current `catalog_items` values are exported at run time, not a snapshot (REQ-015) |

Deferred to a later pass: month/date-range filtering, other formats
(XLSX, JSON), scheduled or emailed reports, admin/cross-user export,
export history/audit log, and any UI trigger.
