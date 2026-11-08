---
feature_code: FEAT-003
service: sandbox-svc
version: 1
status: complete
spec_version: 1
code_repo: https://github.com/william22913/sandbox-svc-code
release_branch: main
integration_branch: feat/FEAT-003
stack: go
dev_feedback: []
code_prep_feedback: []
sources: []
pr_url: https://github.com/william22913/sandbox-docs/pull/19
last_updated: 2026-11-06
---

# FEAT-003 — Export my favorites to CSV (sandbox-svc) — Architecture

## Revision notes (spec v1)

First design pass for this feature. spec.md v1 is complete and merged
(PR 17); testcase.md v1 is complete and merged (PR 18) and pins
`spec_version: 1`. All 21 requirements are covered below.

The feature itself is small: one read-only `GET` route, no new table,
no UI. The design work is concentrated in three places where the
obvious implementation would be subtly wrong:

1. **The response cannot go through `WrapService`.** nexcommon's
   response writer hardcodes `Content-Type: application/octet-stream`
   and an *unquoted* `Content-Disposition` for `model.FileURLPath`
   payloads, and forces `application/json` for everything else.
   REQ-002 requires `text/csv` and a quoted filename, so the export is
   served by a **raw `mux` route** registered on the same router.
2. **The CSV writer cannot be `encoding/csv` with `UseCRLF: true`.**
   Go's writer translates a lone `\n` inside a quoted field into
   `\r\n` and drops a lone `\r`. REQ-009 says field content is
   preserved and only the *record terminator* is CRLF; TC-023 asserts
   embedded CR/LF are verbatim. A ~20-line RFC 4180 writer is used
   instead (see Components).
3. **Three columns do not exist yet.** `category`, `price` and `note`
   are added by this feature — see `database.md`, which is the
   authority for the schema and records the `note`/FEAT-002 split.

## Survey notes (Part 2)

`sandbox-svc-code` was surveyed read-only at `feat/FEAT-001` (PR 1,
merged), because `main` still contains only the repository README.
That branch is the feature's real starting point for `/go-dev`.

What the survey established, and what this design therefore reuses
rather than invents:

- **Stack** — Go 1.27, gorilla/mux, Postgres via `database/sql`+pgx,
  sql-migrate, `nexcommon v0.3.68-swagger`.
- **Route prefix** — `router.V1PrefixPath = "/v1"`. `favorite_endpoint`
  builds `Path = router.V1PrefixPath + "/favorites"`. FEAT-001's
  architecture.md writes unprefixed paths as shorthand; the real paths
  are `/v1/...`. This document uses the real paths only.
- **Existing routes** — `POST /v1/favorites`, `GET /v1/favorites`,
  `DELETE /v1/favorites/{item_id}`, plus `/v1/health` and `/metrics`
  registered directly in `router.InitHttpService`.
- **Auth** — `httpValidator.UserAccessValidator()` over Redis, which
  on a missing/unknown token returns `errors.ErrUnauthorized`
  (`401`, code `E-1-CMD-AUT-001`). It populates
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`.
- **Error envelope (verified against the vendored library, not
  assumed)** — `errors.NewErrorFormator(...).ReformatErrorMessage(
  errors.NewErrorMessageParam(err).WithContext(ctx))` returns
  `{"success": false, "header": {"request_id","version","timestamp"},
  "payload": {"status", "code", "message"}}`. Unknown errors become
  `500` with the project's `defaultInternalCode`,
  `E-5-SBX-SRV-001`.
- **Metrics** — `middleware.StatusAwareMetricsMiddleware` is installed
  with `router.Use(metricMiddleware.Serve)` and reads the real response
  status; `/metrics` is served outside token validation.
- **Existing DAO** — `favorite_dao.List` already selects
  `item_id, name, is_available, created_at` from
  `favorite_items LEFT JOIN catalog_items`, ordered
  `created_at DESC, item_id ASC`. The export reuses that shape
  verbatim so its values match `GET /favorites` (REQ-019/TC-005).

Survey findings recorded but **not** acted on here: FEAT-001's
`architecture.md` documents an envelope
(`{"error_code": "...", "message": "..."}`) that the library does not
emit — its own `docs/FEAT-001/dev-test-results.md` already logs the
correction. FEAT-001's files are outside this role's write scope and
are not edited by this PR; FEAT-003 does not repeat the mistake.

## Frontend

**No UI scope.** `spec.md` declares `has_ui: false`, and its Non-Goals
state explicitly that triggering the export is backend/API only. There
is therefore no `frontend_repo` / `frontend_base_branch` in this
document's frontmatter, no screen, no component and no state design in
this feature. Any future UI trigger is a separate feature with its own
spec and its own `/fe-dev` design; nothing here presumes one.

## Stack / Integration decisions

- **Stack: Go, `nexcommon`** — unchanged from FEAT-001; this feature
  adds no dependency. No new third-party library is introduced: the
  CSV writer is hand-rolled (the standard library's behaviour is wrong
  for REQ-009, see above) and everything else is already vendored.
- **Persistence: Postgres, reuse FEAT-001's tables.** No new table, no
  new datastore, no caching layer. The export reads the live join at
  request time (REQ-015) — deliberately no snapshot and no cache.
- **Auth/session: reused, and verified to exist.** Unlike a greenfield
  assumption, the mechanism is present in the surveyed branch:
  `UserAccessValidator()` + Redis token store + `ContextModel`. The
  raw route calls the *same* validator function the wrapped routes
  use, so anonymous callers get the identical 401 before any data is
  read.
- **Response transport: raw `mux` route.** Chosen over
  `model.FileURLPath` (wrong headers) and over a `WrapService`
  endpoint returning a pre-encoded body (still forced to
  `application/json`). The route is registered in
  `router.InitHttpService` on the same `*mux.Router`, so it inherits
  `router.Use(metricMiddleware.Serve)` and needs no separate metrics
  wiring.
- **Swagger/OpenAPI: hand-authored for this route.** nexcommon's
  generator only sees routes registered through `HandleFunc`, so the
  raw route will not appear in the runtime `/inspect` docs or in a
  regenerated `api/openapi.yaml`. `/go-dev` adds the path to
  `api/openapi.yaml` by hand (and the request to the Postman
  collection); the file already carries a header comment saying the
  response side is authored on top of generated paths.

## Components

New or changed components, all in `sandbox-svc-code`:

- **`dao/favorite_dao.go` — `ListForExport(userID int64)`** (changed
  file, new method). One query, mirroring the existing `List`:
  `favorite_items LEFT JOIN catalog_items`, `COALESCE` fallbacks,
  ordered `f.created_at DESC, f.item_id ASC`. Selects the seven CSV
  values, with `price` read as `COALESCE(trim_scale(c.price), 0)::text`
  and `note` as a nullable string. No `WHERE` clause other than
  `f.user_id = $1`.
- **`services/favorite/export_favorite.go`** (new file).
  - `ExportCSV(ctx) ([]byte, error)` — reads the context user id, calls
    the DAO, and renders the file. Bounded at 500 rows (FEAT-001's
    cap), so the whole file is built in memory; no temp file, no
    streaming, no file-cleanup path.
  - `buildCSV(rows, exportDate) []byte` — pure, DB-free, unit-testable.
    Owns the BOM, the header literal, the CRLF terminator, RFC 4180
    quoting, the `Tersedia`/`Tidak Tersedia` labels, the
    `YYYY-MM-DD` UTC date and the formula neutralization. Splitting it
    out is what makes TC-004–TC-027 testable without a database.
- **`endpoint/favorite_endpoint/export_favorite.go`** (new file) —
  `NewExportHandler(httpValidator, srv, formator) http.HandlerFunc`.
  The raw handler: build a `ContextModel`, call
  `httpValidator.UserAccessValidator().Func(ctx.ToContext(), headers)`
  with headers from `httpValidator.Converter(r)`, map any error through
  the shared `formator` to the standard envelope and return it with
  `Content-Type: application/json`, then on success set
  `Content-Type: text/csv`, a quoted `Content-Disposition`, and
  `Cache-Control: no-store`, and write the bytes.
- **`router/controller.go`** (changed) — adds
  `const FavoriteExportPath = V1PrefixPath + "/favorites/export"` and
  registers `router.HandleFunc(FavoriteExportPath, exportHandler)
  .Methods(http.MethodGet)` inside `InitHttpService`, next to
  `/v1/health`. The path constant lives in `router` (not
  `favorite_endpoint`) because `favorite_endpoint` already imports
  `router`; the reverse import would be a cycle.
- **`server_config/server_attribute.go`** (changed) — builds the
  export handler and passes it to `InitHttpService`, alongside the
  existing `s.errorFormator`.
- **`sql_migrations/main/3_export_columns.sql`** (new) — the single
  batch from `database.md`.
- **`api/openapi.yaml`** and `postman/` (changed) — the hand-authored
  path and request described above.

Untouched: every existing route and service method, the auth/token
stack, the metrics middleware, and the catalog/auth stand-in tables'
existing columns.

**Why the CSV writer is hand-rolled.** `encoding/csv` with
`UseCRLF: true` writes `\r\n` record terminators as required, but while
quoting a field it drops a lone `\r` and rewrites a lone `\n` as
`\r\n`. REQ-009 requires the *field content* to survive unchanged and
only the record terminator to be CRLF, and TC-023 asserts embedded
CR/LF are byte-preserved. A ten-line field writer — quote when the
field contains `,`, `"`, `\r` or `\n`, double embedded quotes — gives
exactly RFC 4180 with no content mutation, and is more predictable
than wrapping the stdlib writer.

**Backend authorization** (required regardless of UI scope):

- Every request must carry a valid user token. Enforced by the reused
  `UserAccessValidator()` before the DAO is touched; anonymous or
  invalid tokens get `401` and no file (REQ-012).
- **The user id is read only from
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`.** The handler
  reads no query parameter, no path parameter, no body field and no
  `X-User-Id`-style header. `?user_id=`, `?userId=` and an `X-User-Id`
  header are simply ignored — they are not bound to anything
  (REQ-013, TC-030).
- There is no role/permission distinction beyond "authenticated", no
  admin path, and no cross-user export (spec Non-Goals). Ownership is
  satisfied *by construction*, exactly as FEAT-001's REQ-008 records:
  the only `WHERE` clause is `f.user_id = <context user id>`, so
  "another user's row" and "no row" are the same zero-row result.
- No filter-options endpoints exist because the endpoint has no
  filters (REQ-003 forbids paging and filtering); the
  filter-data-sourcing principle has nothing to apply to.

## API contract (consumer-facing)

| Method | Path | Consumer | Success | Errors |
|---|---|---|---|---|
| `GET` | `/v1/favorites/export` | `external` | `200`, `Content-Type: text/csv`, `Content-Disposition: attachment; filename="favorites-<user_id>-<YYYY-MM-DD>.csv"`, body = UTF-8 BOM + header + rows | `401` standard envelope (anonymous/invalid token); `500` standard envelope (`E-5-SBX-SRV-001`) on unexpected failure. No `404` — an empty list is a valid `200` (REQ-011). |

Success headers, exactly:

- `Content-Type: text/csv` (no `charset` parameter — the BOM is the
  charset signal the spec chose, and REQ-002 names the bare type).
- `Content-Disposition: attachment; filename="favorites-<user_id>-<YYYY-MM-DD>.csv"`,
  quoted, date from `time.Now().UTC()`, user id from the auth context.
  Both components are machine-generated, so there is no
  header-injection surface.
- `Cache-Control: no-store` — the body is one user's personal data;
  it should not be retained by an intermediary cache.

Error envelope (verified against the vendored library):

```json
{"success": false,
 "header": {"request_id": "...", "version": "...", "timestamp": "..."},
 "payload": {"status": 401, "code": "E-1-CMD-AUT-001", "message": "..."}}
```

The HTTP status is taken from `payload.status` of the formatted
response — the raw handler does not re-derive it from the error.

**No request body.** `GET` with an empty body; anything sent is
ignored (the global `http.MaxBytesReader` still caps it). The endpoint
declares no query parameters, which is itself the answer to REQ-003's
"no paging and no filter": `?page=` and `?category=` are unbound and
have no effect (TC-009).

## Backend — Endpoints

- `GET /v1/favorites/export` — `consumer: external`. Returns the
  calling user's own favorites as a `text/csv` attachment. No request
  parameters, no body. Registered as a raw `mux` route, authenticated
  by the shared `UserAccessValidator()`.

This is the whole API surface this feature adds; no existing endpoint
changes.

## Data design

See `database.md` for the full schema-designer output. Summary: no new
table. `catalog_items` gains `category TEXT NOT NULL DEFAULT ''` and
`price NUMERIC NOT NULL DEFAULT 0`; `favorite_items` gains
`note TEXT NULL`. One additive batch
(`sql_migrations/main/3_export_columns.sql`), no backfill, no index, no
constraint changes. `note` is added here but its write path, trimming
and 200-character validation remain FEAT-002's, which must not re-add
the column.

## Spec traceability

- **REQ-001** — `GET /v1/favorites/export` raw route -> `favorite.Service.ExportCSV` -> `favorite_dao.ListForExport` -> `200` with a CSV body. Covered.
- **REQ-002** — Handler sets `Content-Type: text/csv` and a quoted `Content-Disposition: attachment; filename="favorites-<user_id>-<YYYY-MM-DD>.csv"`; the date is `time.Now().UTC().Format("2006-01-02")`, never server-local. Covered.
- **REQ-003** — One DAO query, no `LIMIT`/`OFFSET`, no filter parameters bound; ordered `f.created_at DESC, f.item_id ASC`, identical to FEAT-001's `List` (TC-008's tie case included). The 500-per-user cap is FEAT-001's, so the result is already bounded. Covered.
- **REQ-004** — The header is a single constant literal `item_id,name,category,price,available,favorited_at,note`, written once before any row. Covered.
- **REQ-005** — `price` is read as `COALESCE(trim_scale(c.price), 0)::text` and written verbatim; no symbol, no grouping, no rounding anywhere in the pipeline. `trim_scale` needs PostgreSQL 13+ and the repo's compose pins `postgres:16-alpine`. Covered.
- **REQ-006** — `created_at` is formatted in Go with `.UTC().Format("2006-01-02")`, so the DB session time zone cannot shift the date (TC-003/TC-014). Covered.
- **REQ-007** — `available` is `"Tersedia"` when `is_available` is true, `"Tidak Tersedia"` otherwise, in `buildCSV`. Covered.
- **REQ-008** — `note` is scanned as nullable (`sql.NullString`); both `NULL` and `''` render as an empty field. The export does not trim, normalize or validate — that is FEAT-002's write-path job. Covered.
- **REQ-009** — Hand-rolled RFC 4180 writer: BOM `EF BB BF` first, CRLF record terminator, comma delimiter, a field quoted when it contains `,`, `"`, `\r` or `\n`, embedded `"` doubled, and every other byte preserved (this is the reason `encoding/csv`+`UseCRLF` is not used). Covered.
- **REQ-010** — `buildCSV` prefixes a single `'` to `name`, `category` or `note` when the value's **first** character is `=`, `+`, `-` or `@`. Applied before quoting, so the neutralized value is what gets RFC 4180-escaped. A value whose first character is a space is out of scope (TC-026). The spec fixes the property, not the character; `'` is the recorded choice. Covered.
- **REQ-011** — Zero rows still writes the BOM + header + CRLF and returns `200`; there is no empty-body or `204` branch. Covered.
- **REQ-012** — The reused `UserAccessValidator()` returns `errors.ErrUnauthorized`; the handler formats it through the shared `error.Formator` and writes the standard envelope with `Content-Type: application/json`. It never sets `text/csv` or `Content-Disposition`. Covered.
- **REQ-013** — The user id comes only from `ctx.AuthAccessTokenModel.AuthenticationServerUserID`; nothing else is bound or read. Covered.
- **REQ-014** — The `LEFT JOIN` keeps the row when the item is unavailable; `is_available = false` renders `Tidak Tersedia`. Rows are never dropped for availability. Covered.
- **REQ-015** — The query runs at request time against the live `catalog_items`; no cache, no snapshot column, no denormalized copy (see `database.md` Phase 2 item 6). Covered.
- **REQ-016** — Uses FEAT-001's `favorite_items`, its live join, and its ownership rule (scoped `WHERE user_id = $1`, no caller-supplied id). Covered.
- **REQ-017** — `favorite_items.note` is added by this feature's migration and read here; the write path is FEAT-002's and does not exist yet, so `NULL` is the only state today. Covered.
- **REQ-018** — The shared auth middleware exists in the surveyed branch (`UserAccessValidator()` + Redis) and is called directly by the raw handler; it is reused, not assumed. Covered.
- **REQ-019** — N favorites -> 1 header + N rows, newest-first, values identical to `GET /favorites` because the query mirrors FEAT-001's `List` join and columns. Covered.
- **REQ-020** — Zero favorites -> header-only CSV with `200` (same code path as REQ-011). Covered.
- **REQ-021** — No request input can select another user: no parameter, header or body field is read for identity (TC-030/TC-031). Covered.

Non-goals, all recorded as not applicable: no UI trigger, no
date-range filtering, no XLSX/JSON/PDF, no scheduled or emailed
export, no admin/cross-user export, no export history, no writes.

## Planned breaking changes

None. This feature adds one route and three columns; no existing
endpoint, response shape or column changes, and nothing an existing
consumer depends on is removed or renamed.

## Implementation Plan

- PART-1: Schema migration — `sql_migrations/main/3_export_columns.sql`, the single additive batch in `database.md` (category/price on `catalog_items`, note on `favorite_items`), with up/down sections and the batch's validation checks.
- PART-2: DAO and CSV rendering — `favorite_dao.ListForExport` plus `services/favorite/export_favorite.go` (`ExportCSV` and the pure `buildCSV`), with unit tests for the header, ordering, price/date/availability formatting, NULL note, RFC 4180 quoting, BOM/CRLF and formula neutralization.
- PART-3: Raw HTTP route — `favorite_endpoint.NewExportHandler`, the path constant and registration in `router/controller.go`, composition in `server_attribute.go`, and handler-level tests for the 200 header set, the 401 envelope, and the zero-row case.
- PART-4: Contract artifacts — hand-author `GET /v1/favorites/export` in `api/openapi.yaml` and add the request to `postman/`, noting the route is invisible to nexcommon's generator.

PART-2 depends on PART-1's columns; PART-3 depends on PART-2; PART-4
depends on PART-3. There is no frontend part.

## Observability

- **Current state** — the surveyed branch already has the full metrics
  stack: `/metrics` exposed in `router.InitHttpService` outside token
  validation, `middleware.StatusAwareMetricsMiddleware` installed on
  the router, and nexcommon `v0.3.68` whose writer no longer hardcodes
  status 200 (the reproduced middleware reads the captured status).
  Nothing needs wiring.
- **Decision** — **reuse** the existing instrumentation; add no
  collector, no registry and no dashboard. The new route sits on the
  same `*mux.Router`, so `router.Use(metricMiddleware.Serve)` observes
  it with no extra code. No feature-specific metric is needed: the
  spec states there is no post-launch business metric for this
  feature, and the standard request metric already distinguishes a
  `200` export from a `401`.
- **Implementation boundary** — inbound HTTP only, on
  `GET /v1/favorites/export`. The label set stays bounded: the
  middleware resolves the template via
  `mux.CurrentRoute(r).GetPathTemplate()`, which for this route is the
  literal `/v1/favorites/export` — no ids, no user ids, no query
  strings in labels. `/metrics` keeps its existing deployment-level
  access restriction.
- **Verification** — during implementation, scrape `/metrics` after an
  authenticated export and confirm a `200` sample for the route
  template, then after an anonymous call confirm a `401` sample for
  the same route (not folded into `200`), and confirm `/metrics` does
  not count itself. Because the middleware is shared rather than
  per-feature, also confirm the existing `GET /v1/favorites` samples
  are not duplicated by this feature's route registration.

## Documented deviations from the spec's wording

Recorded here so a later docs pass can correct the source wording.
**No blocking `code_prep_feedback` entry is filed for these** — the
user explicitly asked that they be documented in the design instead of
blocking the PR.

1. **Path prefix.** spec.md and testcase.md write `GET /favorites/export`.
   The real route is **`GET /v1/favorites/export`** — the repo's own
   convention (`router.V1PrefixPath`), which FEAT-001's code already
   applies. Every testcase still applies verbatim to the real path.
   Recommended fix: a later `/document-preparation` pass should write
   the prefixed path in FEAT-003's spec.md.
2. **401 error code.** spec.md REQ-012 and TC-028/TC-029 describe the
   envelope as `UNAUTHENTICATED`. The implemented library returns the
   real code **`E-1-CMD-AUT-001`** (nexcommon's `ErrUnauthorized`,
   `status: 401`) with a translated message. The envelope shape is the
   `{success, header, payload{status, code, message}}` form, not a
   top-level `error_code`/`message` pair. Recommended fix: a later
   `/testcase-generation` / `/document-preparation` pass should name
   `E-1-CMD-AUT-001` (or assert only the `401` status and "standard
   unauthenticated envelope") instead of a literal `UNAUTHENTICATED`.
3. **`note` provenance.** spec.md REQ-017 states the dependency on
   FEAT-002's `note` field as if it exists. It does not — FEAT-002 has
   no architecture, database or code yet. Per the user's decision,
   FEAT-003's migration creates the nullable column and reads it;
   FEAT-002 later owns writes and validation. Recorded in
   `database.md` as well, because FEAT-002's own migration must not
   re-add the column.
