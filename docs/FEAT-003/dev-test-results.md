# FEAT-003 — Dev Test Results

Written by `/go-dev` for `sandbox-svc` / `FEAT-003`. One row per test case with
its evidence, then the live-stack run, the deliberate deviations, and the parts
of this pass that could not be exercised.

| | |
|---|---|
| Spec version | `1` — `docs/FEAT-003/spec.md` (merged, docs PR 17) |
| Testcase version | `1` — `docs/FEAT-003/testcase.md` (merged, docs PR 18) |
| Architecture version | `1` — `docs/FEAT-003/architecture.md` (merged, docs PR 19) |
| Database version | `1` — `docs/FEAT-003/database.md` |
| Part PRs | PART-1…PART-4 are delivered together in **one** PR into `feat/FEAT-003`, from branch `feat/FEAT-003-dev-export-favorites-csv`. The PR is opened after this document exists; its title names PART-1…PART-4 and its body lists them. |
| Branch base | `feat/FEAT-001` — the starting point `architecture.md` names |

## How to run

```sh
go build -mod=vendor ./...
go test -mod=vendor -count=1 ./tests/...
```

This machine has no Go installation, so those exact commands run inside the
`golang:1.27.2-alpine` container that `docker-compose.test.yml` starts — gofmt,
vet and test in one pass, and the output below is from that container:

That run printed `gofmt: clean`, `go vet` with no output, and `ok` for all six
packages (`tests`, `tests/db`, `tests/dto`, `tests/endpoint`, `tests/middleware`,
`tests/service`). `tests/db` prints its own SKIP line — see Known limitations.

The live stack is verified with the runner's `verify_stack` tool — the only way
to run Docker here. Its parameters, as used for the run below:

```
repo_path: work/code
compose_override: docker-compose.verify.yml
newman_collection: postman/collection.json
newman_environment: postman/environment.verify.json
```

## Implementation points

| # | Point | Files | Tests |
|---|---|---|---|
| PART-1 | Additive schema batch (category/price on `catalog_items`, note on `favorite_items`) | `sql_migrations/main/3_export_columns.sql` | exercised by `tests/db` and by the live stack |
| PART-2 | Export query + CSV renderer | `dao/favorite_dao.go` (`ListForExport`), `dao/type.go` (`FavoriteExportItem`), `services/favorite/export_favorite.go` | `tests/service/export_favorite_test.go`, `tests/db/favorite_export_dao_test.go` |
| PART-3 | Raw HTTP route | `endpoint/favorite_endpoint/export_favorite.go`, `router/controller.go`, `server_config/{type,server_attribute}.go`, `main.go` | `tests/endpoint/export_favorite_test.go`, `tests/endpoint/favorite_endpoint_test.go` |
| PART-4 | Contract artifacts | `api/openapi.yaml`, `postman/collection.json`, `postman/enrich-collection.js`, `postman/environment.verify.json`, `docker-compose.verify.yml`, `docker-compose.test.yml`, `tests/live/*` | the Newman run below |

## Test case results

`live` means the assertion ran against the real stack through Newman; anything
else named is a Go test function (all of which appear in the run above).

| Case ID | Result | Evidence | Notes |
|---|---|---|---|
| TC-001 | PASS | live: `Export favorites as CSV` — 200, `text/csv`, non-empty body starting with the header | Body compared byte for byte after the BOM. |
| TC-002 | PASS | `TestExportFileNameUsesUTCDate_TC002_TC003` (case 1); live: `Content-Disposition` matches `attachment; filename="favorites-<id>-<UTC date>.csv"` | The id comes from the auth context, not the request. |
| TC-003 | PASS | `TestExportFileNameUsesUTCDate_TC002_TC003` cases 2–3 (23:59:59Z and 00:00:01Z expressed in Jakarta) | Unit-level: the live stack runs in UTC, so the zone boundary cannot be reproduced there. |
| TC-004 | PASS | `TestExportCSVHeaderBOMAndCRLF_TC004_TC020`; live: exact body assertion starts at the header row | |
| TC-005 | PASS | `TestExportCSVKeepsDAOOrder_TC005_TC008`; live body order 103, 102, 101, 104, 105 = newest-favorited first | The export and `GET /favorites` share one `ORDER BY`; the two responses were not diffed against each other for the same user. |
| TC-006 | PASS (unit) | `TestExportCSVRendersEveryRow_TC006_TC007` — 500 rows, 501 records, no truncation | A 500-row live fixture would make every other assertion unreadable. |
| TC-007 | PASS (unit) | same test — the one-favorite export | |
| TC-008 | PASS (unit) | `TestExportCSVKeepsDAOOrder_TC005_TC008` — three rows with the *same* instant keep DAO order | The tie-break itself (`ORDER BY created_at DESC, item_id ASC`) is SQL, covered by `tests/db`, which skipped here — see Known limitations. |
| TC-009 | PASS | live: `?page=2&category=Office` returns all five rows, byte-identical to the unfiltered body | The route binds no query parameters. |
| TC-010 | PASS | `TestExportCSVPriceVerbatim_TC010_TC011_TC012`; live: cell `1500` | |
| TC-011 | PASS | same test; live: cell `19.99` | Canonicalization of a stored `19.9900` is proved by `TestListForExportCanonicalizesPriceScale` (SQL, skipped here). |
| TC-012 | PASS | same test; live: cell `1234.5678`, unrounded | |
| TC-013 | PASS | `TestExportCSVDateIsUTC_TC013_TC014`; live: `2026-04-11` | |
| TC-014 | PASS (unit) | `TestExportCSVDateIsUTC_TC013_TC014` — 2026-02-28T23:59:59Z and 2026-03-01T00:00:00Z | |
| TC-015 | PASS | `TestExportCSVAvailabilityLabels_TC015_TC016`; live: `Tersedia` | |
| TC-016 | PASS | same test; live: the discontinued item 102 is present with `Tidak Tersedia` | Not dropped, not filtered. |
| TC-017 | PASS | `TestExportCSVNoteEmptyCells_TC017_TC018`; live: row 104 ends `favorited_at,` + CRLF | |
| TC-018 | PASS (unit) | same test — stored `''` renders as the same empty cell | |
| TC-019 | PASS | `TestExportCSVUtf8PassThrough_TC019_TC024`; live: `Café — 🎁` in the note cell | Unit case carries the full `Café — 🎁 for mom`. |
| TC-020 | PASS | `TestExportCSVHeaderBOMAndCRLF_TC004_TC020`; live: first three bytes are `ef bb bf`, every record ends CRLF | |
| TC-021 | PASS | `TestExportCSVQuoting_TC021_TC022_TC023`; live: `"Notebook, A5"` | |
| TC-022 | PASS | same test; live: `"He said ""hi"", twice"` | |
| TC-023 | PARTIAL | `TestExportCSVQuoting_TC021_TC022_TC023` — a note containing `line1\r\nline2` is quoted and survives unquoting | Live, an embedded **LF** in a name is preserved inside quotes; there is no CRLF-in-note row in the live fixture. |
| TC-024 | PASS | `TestExportCSVUtf8PassThrough_TC019_TC024` (`Café` / `Bücher` / `日本語`); live: accented Latin, em dash and emoji decode intact | |
| TC-025 | PASS (unit) | `TestExportCSVNeutralizesFormulas_TC025_TC026` — all 12 field × character cells | |
| TC-026 | PASS (unit) | same test — a leading space and a mid-string `=` are left alone | |
| TC-027 | PASS | `TestExportCSVZeroFavoritesIsHeaderOnly_TC027`; live: user 3 gets 200, header-only, `text/csv`, attachment disposition | |
| TC-028 | PASS | `TestExportHandlerRejectsUnauthenticated_TC028_TC029`; live: anonymous 401, `E-1-CMD-AUT-001`, no `Content-Disposition`, JSON envelope | |
| TC-029 | PASS | same test (invalid token case); live: same envelope for an unseeded token | |
| TC-030 | PASS | live: user 3 sends `?user_id=1` **and** `X-User-Id: 1` and still receives only their own (empty) file | The handler reads no query parameter and no header other than `Authorization`. |
| TC-031 | PASS | live: user 1's file has exactly five rows, user 3's has none; disjoint | |
| TC-032 | PARTIAL | The schema makes a snapshot impossible: `favorite_items` has no name/price/availability columns, and `ListForExport` joins `catalog_items` live. `TestListForExportReadsLiveValuesAndNullableNote` asserts an update-then-read, but it is in `tests/db` and skipped here (no docker socket). | Marked partial rather than PASS: the live fixture cannot distinguish live values from a snapshot because nothing mutates between seeding and export. |

## Live verification

`verify_stack` returned **PASS**: all five services healthy (app, postgres,
redis, and the two seed sidecars), Newman exit 0, **8 requests / 13 test
scripts / 32 assertions / 0 failures**.

| Check | Case | Result |
|---|---|---|
| Exact CSV file for a five-row user, BOM asserted as bytes | TC-004, TC-005, TC-010…TC-024 | Body equals the expected file byte for byte |
| Unsupported query parameters change nothing | TC-009 | All rows still returned |
| Caller-supplied id in query and header ignored | TC-030 | Only the caller's own rows |
| Second user's file is header-only | TC-031, TC-027 | 200, header only |
| Anonymous and invalid token | TC-028, TC-029 | 401 `E-1-CMD-AUT-001`, no file headers |
| `Content-Disposition` shape | TC-002 | `attachment; filename="favorites-<id>-<UTC date>.csv"` |

### Observability

`architecture.md` decided to reuse the existing status-aware middleware rather
than add a metric. Verified on the live stack by scraping `/metrics` after the
requests above:

- `my_app_http_hist_count{method="GET",path="/v1/favorites/export",status="200"}` exists — the raw route is observed under its own template, not folded into `/v1/favorites`.
- `my_app_http_hist_count{...,status="401"}` exists for the same path — the rejected export is recorded with its real status.
- No `path="/metrics"` series — the scrape does not count itself.

## Deviations from the feature docs

1. **`buildCSV` takes only the rows.** `architecture.md` sketched
   `buildCSV(rows, exportDate)`. No CSV column contains the export date — the
   date appears only in the filename, which `endpoint.ExportFileName` builds —
   so the parameter would have been dead. Dropped deliberately, taken as a
   documented deviation rather than a silent one.
2. **The export route is not in the generated swagger.** It is registered
   directly on the mux router because the response is a file with its own
   headers, and nexcommon's wrapper forces `application/json`. The route is
   therefore hand-authored in `api/openapi.yaml` and exempted — explicitly, with
   a positive "must be documented" assertion — in
   `TestOpenAPISpecMatchesRegisteredRoutes`.
3. **The generated Postman request for the export is replaced.** The
   converter emits one request for the path that inherits the collection's
   single token; it cannot cover "another user", "anonymous" and "invalid
   token" in one run. `postman/enrich-collection.js` drops that one generated
   request and injects four with their own auth, plus the metrics scrape. The
   collection stays a build artifact — regenerate, then run the script.
4. **`tests/live/` and the two verify compose files are not shipped code.**
   `docker-compose.verify.yml`, `docker-compose.test.yml` and the seed scripts
   exist only for the runner; the shipped `docker-compose.yml` and `Dockerfile`
   are unchanged.
5. **DAO context.** `ListForExport` uses `context.Background()`, like every other
   method in the DAO, so a client disconnect does not cancel the query. Raised
   by the SQL security gate as a minor, pre-existing pattern; changing it means
   threading a context through the service and DAO interfaces for all methods,
   which is a refactor of FEAT-001 code, not of this feature.

## Known limitations

- **`tests/db` did not execute.** It starts its own Postgres through the Docker
  CLI; the Go test container has no Docker socket, so `testdb.Start` returns
  `ErrNoDocker` and the package prints SKIP. The same SQL — join, `trim_scale`,
  ordering, nullable note — was exercised end to end by the live stack instead,
  and the exact CSV body it produced is the assertion.
- **No CI workflow exists in this repo**, so there is no CI JUnit report for
  this pass. The evidence above is the local containerised run plus the runner's
  Newman output. Adding CI is outside this feature's scope.
- **TC-023 and TC-032 are marked PARTIAL** for the reasons in the table; both
  have unit-level coverage that did not execute here.
- `verify_bundle_coverage.py` was not required: this feature adds no error
  sentinel and no validator rule key, so the i18n bundles are unchanged.

## Delivery guard

| Check | Result |
|---|---|
| `verify_generated_go_layout.py` | PASS — Node port: `Checked 14 generated file(s) against <repo>. OK - no misplaced endpoint files, no leaked template artifacts.` |
| `verify_bundle_coverage.py` | Not applicable — no new sentinel or rule key |
| `gofmt` / `go vet` / `go test ./...` | PASS in `golang:1.27.2-alpine` (gofmt clean, vet silent, all packages ok) |
| SQL security gate (`sql-security-validator`) | **PASS** — no blockers, no majors. Injection surface: one literal statement, `userID int64` bound as `$1`; no dynamic SQL or identifiers. Scoping: `f.user_id = $1` from the auth context only. DDL: transactional `SET LOCAL lock_timeout`, metadata-only constant defaults, symmetric down. Two minor notes: a fixture comment said "four" where five rows exist (fixed), and the pre-existing `context.Background()` pattern in the DAO (accepted, see deviations). |
| Live stack + Newman | PASS — 8 requests, 32 assertions, 0 failures |
| Observability | PASS — export observed as its own `path` label with real `status` values |
