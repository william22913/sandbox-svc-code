---
feature_code: FEAT-005
service: sandbox-svc
spec_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — Favorites Summary Tooltip (PRD)

## Overview

FEAT-004's header badge shows only a total favorite count. FEAT-005 adds a
per-user summary endpoint and a hover tooltip on that same badge, so a
signed-in user can see how many of their saved items are discontinued
without opening "My Favorites". See [spec.md](spec.md) for the full
requirement detail.

## Product Goal

Give signed-in users an at-a-glance breakdown of their saved items — total
favorites and how many of them are discontinued — through the existing
FEAT-004 badge, without adding click-through, realtime updates, caching, or
any new storage in this pass.

## Features

| Feature | Description | Expected behavior |
|---|---|---|
| Summary endpoint | `GET /v1/favorites/summary` | Returns `200 {"total": T, "discontinued": D}` for the authenticated user's own favorites; `total` includes discontinued items (FEAT-001 REQ-004), and `discontinued` counts favorites whose catalog item is currently soft-marked discontinued; zero favorites returns `{0,0}` (REQ-001, REQ-002, REQ-003, REQ-004, REQ-012) |
| Ownership and access | Whose favorites are summarized | Counts always come from the auth context; no caller-supplied user id; anonymous callers get `401` (REQ-010, REQ-011, REQ-013) |
| Badge tooltip | Hover text on the FEAT-004 badge | Exact text `N favorites (M discontinued)`, with exact integers even when the badge itself shows `99+`; shown only when the badge is visible (REQ-005, REQ-006, REQ-015) |
| Fetch and refresh | When the summary loads | Fetched on first hover, not on app load; an already-fetched summary is refreshed after a successful favorite/un-favorite; pre-action values are never shown (REQ-007, REQ-018) |
| Hidden fallback states | Loading, error, unauthenticated | Tooltip is hidden while the summary loads or after a failed/`500` fetch; the badge keeps its FEAT-004 count display unchanged (REQ-008, REQ-016, REQ-017) |

Deferred to a later pass: click-through, polling/realtime updates, caching,
per-item breakdown, analytics, localization/pluralization, and tooltip
animation.
