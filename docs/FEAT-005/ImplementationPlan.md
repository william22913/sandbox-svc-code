---
feature_code: FEAT-005
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — Implementation Plan Overview

> Summary only — see `architecture.md`'s `## Implementation Plan` section
> for full detail.

## Overview

Two ordered backend parts (no timeline or estimate — order of work only),
followed by a separately sequenced frontend delivery. The feature has UI
scope, but the frontend is deliberately not a PART ID: `/go-release` maps
every PART here to a merged `/go-dev` PR in `sandbox-svc-code`, and
`/fe-dev` records no PART in `dev-test-results.md`, so a frontend-only
PART could never satisfy that gate.

## Goals

Ship `GET /v1/favorites/summary` as a read-only aggregate of the calling
user's own favorites — `total` and `discontinued`, index-backed, scoped by
the auth context, computed live with no cache or counter column — on the
existing Go + nexcommon + Postgres service, with no schema change; then
extend FEAT-004's badge with a first-hover tooltip that shows
`N favorites (M discontinued)` and is invalidated after each successful
favorite/un-favorite.

## Implementation Plan

| Part | What gets built | Key deliverable |
|---|---|---|
| PART-1 | Summary read path | `dao.FavoriteDAO.Summary` (interface + implementation), `dao.FavoriteSummary`, `services/favorite/summary_favorite.go`, `dto/out.FavoriteSummaryResponse` with `NewFavoriteSummary`, plus unit tests for zero/zero, one/zero, one/one, mixed and all-discontinued rows, the 500-row cap, two-user isolation, a missing catalog row, and the auth-context user id being the only scope. |
| PART-2 | HTTP route and contract | `favorite_endpoint.registerSummary` on `GET <Path>/summary` and its call from `RegisterEndpoint()`, the OpenAPI/Postman artifact for the route, handler tests for the `200` payload, the `401` envelope, the `500` envelope and the `0` case, plus the `/metrics` 200/401 scrape check from `## Observability`. |

PART-2 depends on PART-1. There is no frontend PART: the tooltip work in
`sandbox-svc-ui` (summary state and `ensureSummary()`/`refreshSummary()`
in FEAT-004's `FavoriteCountProvider`, the Ant Design `Tooltip` around the
`AppShell` badge, the summary adapter/service mapper, `refreshSummary()`
calls from `useFavoriteStar` and `useFavorites`, and tests) is sequenced
by `/fe-dev` from `architecture.md`'s `## Frontend` section — after
FEAT-004's badge has landed in the FE repo and PART-2's contract exists.
