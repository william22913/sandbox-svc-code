---
feature_code: FEAT-001
service: sandbox-svc
locked_version: 1
last_updated: 2026-10-10
---

# FEAT-001 — Implementation Plan Overview

> Summary only — see `architecture.md`'s `## Implementation Plan` section
> for full detail.

## Overview

Backend-only build for this pass, sequenced in four ordered phases (no
timeline/estimate — order of work only).

## Goals

Ship favorite/un-favorite and the "My Favorites" list, correctly handling
the idempotency (REQ-005), the 500-favorite cap (REQ-007), and
discontinued items shown as unavailable rather than vanishing (REQ-004),
on a brand-new Go + nexcommon service.

## Implementation Plan

| Phase | What gets built | Key deliverable |
|---|---|---|
| 1 | Data model and migration | `favorite_items` table (composite PK, no soft-delete) per `database.md`'s Batch 1; initial nexcommon/Postgres wiring for this greenfield repo. |
| 2 | Core service logic | `favorite_service`: idempotent add (`ON CONFLICT DO NOTHING`), 500-cap check, remove, and the list-with-availability join. |
| 3 | API endpoints | `favorite_endpoint`: `POST /favorites`, `DELETE /favorites/{item_id}`, `GET /favorites`, wired via nexcommon's controller-wrapping conventions. |
| 4 | Integration/observability wiring | Standard nexcommon request metrics on the three new routes, with status-aware instrumentation covering REQ-007's 422 response distinctly. |

Frontend phase deliberately omitted — out of scope this pass; a later
`/code-preparation` + `/fe-dev` pass adds it once design/Figma unblocks.
