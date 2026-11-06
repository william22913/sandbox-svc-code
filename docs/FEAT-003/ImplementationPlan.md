---
feature_code: FEAT-003
service: sandbox-svc
locked_version: 1
last_updated: 2026-11-06
---

# FEAT-003 — Implementation Plan Overview

> Summary only — see `architecture.md`'s `## Implementation Plan` section
> for full detail.

## Overview

Backend-only build, sequenced in four ordered parts (no timeline or
estimate — order of work only). No frontend part, because the feature
has no UI scope.

## Goals

Ship `GET /v1/favorites/export` as a read-only CSV download of the
calling user's own favorites — correct header, ordering, formatting,
RFC 4180 quoting and formula neutralization — on the existing Go +
nexcommon + Postgres service, with an additive three-column migration
and no change to any existing endpoint.

## Implementation Plan

| Part | What gets built | Key deliverable |
|---|---|---|
| PART-1 | Schema migration | `sql_migrations/main/3_export_columns.sql`: `catalog_items.category`, `catalog_items.price`, `favorite_items.note` in one additive batch, with up/down sections and the batch's validation checks. |
| PART-2 | DAO and CSV rendering | `favorite_dao.ListForExport` plus `services/favorite/export_favorite.go` (`ExportCSV` and the pure `buildCSV`), with unit tests covering header, ordering, price/date/availability formatting, NULL note, RFC 4180 quoting, BOM/CRLF and formula neutralization. |
| PART-3 | Raw HTTP route | `favorite_endpoint.NewExportHandler`, the path constant and registration in `router/controller.go`, composition in `server_attribute.go`; handler tests for the 200 header set, the 401 envelope and the zero-row case. |
| PART-4 | Contract artifacts | `GET /v1/favorites/export` hand-authored in `api/openapi.yaml`, plus the matching request in `postman/`, noting the route is invisible to nexcommon's generator. |

PART-2 depends on PART-1's columns, PART-3 on PART-2, PART-4 on
PART-3.
