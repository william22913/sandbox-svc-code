---
feature_code: FEAT-005
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — Backend Schema Overview

> Summary only — see `database.md` for the full schema-designer record
> and `architecture.md`'s `## Data design` for the query rationale.

## Overview

No new table, column, index or constraint. The feature reads FEAT-001's
existing `favorite_items` joined to FEAT-003's extant `catalog_items` and
changes nothing about either. The schema-designer's three phases are
recorded as explicitly not applicable, no migration file is produced, and
`ERD.md` is unchanged.

## Data Model

- `favorite_items` — read only. Both counts come from one statement:

  ```sql
  SELECT COUNT(*) AS total,
         COUNT(*) FILTER (WHERE NOT COALESCE(c.is_available, FALSE)) AS discontinued
    FROM favorite_items f
    LEFT JOIN catalog_items c ON c.id = f.item_id
   WHERE f.user_id = $1
  ```

- `catalog_items` — read only, through a primary-key lookup per favorite.
  `total` applies no availability predicate, so discontinued favorites
  stay counted; `discontinued` is the subset whose
  `catalog_items.is_available` is false (or whose catalog row is missing,
  via `COALESCE`). The join is live at request time, so an item
  discontinued after being favorited counts as discontinued and one
  re-enabled stops counting.
- `user_id` is the leading column of the composite primary key
  `(user_id, item_id)`, so the predicate is served by the existing PK
  index; no additional index is justified for a set capped at 500 rows
  per user.
- **No denormalized counter.** A `users.favorite_count` /
  `users.discontinued_favorite_count` column would need writes on every
  add/remove, a write on every catalog availability change, a backfill and
  a consistency story — the spec's Non-Goals forbid new columns and
  caching.
- **No snapshot of availability** on `favorite_items`; the current state
  is required, so the live join is kept.
- Unchanged: the composite PK, both foreign keys, `created_at`
  (`favorited_at`), `note`, and `catalog_items` in its entirety. No
  soft-delete or audit columns are introduced, and no migration batch is
  produced.

## Access Control

- Only an authenticated caller may read a summary. The route goes through
  nexcommon's `WrapService` with `UserAccessValidator()`, so anonymous or
  invalid tokens get `401 E-1-CMD-AUT-001` before the service runs.
- **The user id comes only from
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`.** The endpoint
  declares no query, path or body parameters, so no request input can
  select another user's rows.
- No role or permission distinction beyond "authenticated"; no admin or
  cross-user summary exists — the spec's Non-Goals.
- Ownership is satisfied by construction: the only `WHERE` clause is
  `favorite_items.user_id = <context user id>`.
