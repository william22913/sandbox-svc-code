---
feature_code: FEAT-001
service: sandbox-svc
locked_version: 1
last_updated: 2026-10-10
---

# FEAT-001 — Backend Schema Overview

> Summary only — see `database.md` for the full schema-designer output
> (draft ERD, normalization review, batched migration plan).

## Overview

One new table this feature: `favorite_items`, a pure join between a user
and a catalog item, with no additional business-workflow state.

## Data Model

- `favorite_items` — links `users` (existing, owned by auth/session) to
  `catalog_items` (existing, owned by catalog) via a composite primary
  key `(user_id, item_id)`. No surrogate key (nothing ever addresses a
  favorite by any ID other than the pair itself). No soft-delete columns
  — un-favoriting is a literal, permanent row delete, not a lifecycle
  state. Carries only `created_at` (`favorited_at`), kept for a possible
  future recency-sort pass even though sorting is deferred this pass.

## Access Control

- Every row is scoped implicitly to its own `user_id`, taken from the
  authenticated request context — never from a caller-supplied ID.
- No role/permission distinction beyond "authenticated": a user may only
  ever read or mutate their own favorites. No admin or cross-user view
  exists in this pass.
- Anonymous (unauthenticated) requests never reach the data layer — the
  assumed shared auth middleware returns 401 first.
