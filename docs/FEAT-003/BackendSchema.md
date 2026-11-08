---
feature_code: FEAT-003
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-06
---

# FEAT-003 — Backend Schema Overview

> Summary only — see `database.md` for the full schema-designer output
> (draft ERD, normalization review, batched migration plan).

## Overview

No new table. This feature is strictly read-only at runtime and adds
three columns to tables FEAT-001 already created — two item attributes
the CSV must write, and the note it must read. One additive batch
(`sql_migrations/main/3_export_columns.sql`), no backfill, no new index,
no constraint changes.

## Data Model

- `catalog_items` — gains `category TEXT NOT NULL DEFAULT ''` and
  `price NUMERIC NOT NULL DEFAULT 0`. Both are item attributes, so they
  belong on the item, not on each user's favorite row; the export reads
  them live at request time (REQ-015), never as a snapshot. `price` is
  unconstrained `NUMERIC` so `1500`, `19.99` and `1234.5678` survive
  exactly; it is read as `trim_scale(price)::text` to strip redundant
  trailing zeros.
- `favorite_items` — gains `note TEXT NULL`. A note describes one
  user's relationship to one item, so it belongs on the
  `(user_id, item_id)` pairing and is deleted with the favorite row
  automatically. `NULL` is the "no note" state; an empty string renders
  identically in the export. The **write path, trimming and
  200-character validation belong to FEAT-002** — this feature only
  creates and reads the column, and FEAT-002's later migration must not
  re-add it.
- Unchanged: `favorite_items`' composite `(user_id, item_id)` primary
  key, its foreign keys to `users` and `catalog_items`, its `created_at`
  (`favorited_at`), and `users` in its entirety. No soft-delete columns,
  no `updated_at`, no audit columns — FEAT-001's documented convention
  for these tables, deliberately not revisited.
- The export query mirrors FEAT-001's `List` exactly
  (`LEFT JOIN` + `COALESCE`, ordered `created_at DESC, item_id ASC`),
  which is what makes the CSV match `GET /favorites`.

## Access Control

- Only an authenticated caller may export. The route reuses
  nexcommon's `UserAccessValidator()`; anonymous or invalid tokens get
  `401 E-1-CMD-AUT-001` before any query runs.
- **The user id comes only from
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`.** No query
  parameter, path parameter, body field or `X-User-Id`-style header is
  bound, so no request input can select another user's rows.
- No role or permission distinction beyond "authenticated". There is no
  admin path, no cross-user export, and no export history — the spec's
  non-goals.
- Ownership is satisfied by construction: the only `WHERE` clause is
  `favorite_items.user_id = <context user id>`, so another user's row
  and a missing row are the same zero-row result.
