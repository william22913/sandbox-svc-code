# FEAT-005 — Dev Test Results

Written by `/go-dev` for `sandbox-svc` / `FEAT-005`. One row per test case with
its evidence, then the live-stack attempt, the observability check, the
deliberate deviations, and what this environment could not execute.

| | |
|---|---|
| Spec version | `1` — `docs/FEAT-005/spec.md` (merged, docs PR 23) |
| Testcase version | `1` — `docs/FEAT-005/testcase.md` (merged, docs PR 24) |
| Architecture version | `1` — `docs/FEAT-005/architecture.md` (merged, docs PR 25) |
| Database version | `1` — `docs/FEAT-005/database.md` (merged, docs PR 25) |
| Part PRs | PART-1 and PART-2 are delivered together in the `/go-dev` PR opened from `feat/FEAT-005-dev-favorites-summary-endpoint` into `feat/FEAT-005`. |
| Branch base | `feat/FEAT-005` — the integration branch `architecture.md` names |

## How to run

```sh
go build -mod=vendor ./...
go test -mod=vendor -count=1 ./tests/...
```

This machine has no system Go and no Docker socket. The commands above were run
with the session's Go 1.27.3 toolchain against the vendored module. The run was
clean: `gofmt` reported nothing, `go vet ./...` printed nothing, and every test
package returned `ok`.

`tests/db` prints its own `SKIP: docker not available` line and exits 0 —
database-backed tests are compiled but not executed here (see Known
limitations).

The live stack is verified with the runner's `verify_stack` tool — the only way
to run Docker in this environment. Its parameters, as used:

```
repo_path:         /…/work/code   (absolute; see Known limitations)
compose_file:      docker-compose.yml
compose_override:  docker-compose.verify.yml
newman_collection: postman/collection.json
newman_environment: postman/environment.verify.json
```

## Implementation points

| # | Point | Files | Tests |
|---|---|---|---|
| PART-1 | Summary read path: aggregate query, DAO row type, service, response DTO | `dao/favorite_dao.go` (`Summary`), `dao/type.go` (`FavoriteSummary`), `services/favorite/summary_favorite.go`, `dto/out/favorite.go` | `tests/service/summary_service_test.go`, `tests/dto/favorite_dto_test.go`, `tests/db/favorite_dao_test.go` |
| PART-2 | HTTP route and contract: `GET <Path>/summary`, OpenAPI/Postman artifact, handler and metrics checks | `endpoint/favorite_endpoint/favorite_endpoint.go` (`registerSummary`), `api/openapi.yaml`, `postman/collection.json`, `postman/enrich-collection.js`, `postman/environment.verify.json` | `tests/endpoint/summary_endpoint_test.go`, `tests/endpoint/favorite_endpoint_test.go` |

Both points are implemented in this session. `database.md` confirmed zero
migration batches, so no schema file changed.

## Test case results

`live` below means the assertion ran against the real stack through Newman,
against the seeded fixture. Other evidence named in a row is a Go test that
executed here, except where marked *skipped* — `tests/db` needs the Docker CLI,
which this environment does not expose.

The handler tests drive the real registered route, the real
`UserAccessValidator` → Redis → context path (against an in-process RESP2
server), the real DTO serialisation and the real error formator. The DAO is a
fake, so they prove the HTTP contract and the identity/derivation rules but not
the SQL.

| Case ID | Result | Evidence | Notes |
|---|---|---|---|
| TC-001 | PASS | live: `Summary — user 1` → 200, envelope integer-exact `{5,1}`; `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued`; `TestFavoriteSummaryMarshalsExactIntegers` | The aggregate ran against Postgres here; item 102 is the seeded discontinued row. |
| TC-002 | PASS | live: `Summary — user 3 (zero favorites)` → 200, `{0,0}`; `TestSummaryHandler_TC002_ZeroFavoritesIsA200` | Not 404/204/empty: integer zeros in the usual envelope. |
| TC-003 | BLOCKED | `TestSummary_TC003_SingleAvailableFavorite` (service mapping) passes; `tests/db/TestSummary_TC003_SingleAvailableFavorite` **skipped** | No single-available-favorite user in the live fixture. The SQL boundary did not execute. |
| TC-004 | PASS | live: `Summary — user 1` → `{5,1}` — five owned rows, one of them discontinued; `tests/db/TestSummary_TC004_MixedPartition` skipped | The `COUNT(*) FILTER` partition over an unpartitioned row set. |
| TC-005 | BLOCKED | service test passes; `tests/db/TestSummary_TC005_AllDiscontinuedEqual` skipped | No all-discontinued user in the live fixture; total == discontinued is only provable against a seeded one. |
| TC-006 | BLOCKED | service test passes; `tests/db/TestSummary_TC006_FiveHundredCap` skipped | The fixture holds five favorites, not 500. |
| TC-007 | BLOCKED | service test passes; `tests/db/TestSummary_TC007_AboveNinetyNineNotCapped` skipped | Same fixture limit. Nothing in the code clamps, but that is not live evidence. |
| TC-008 | BLOCKED | service test passes; `tests/db/TestSummary_TC008_SingleDiscontinued` skipped | No single-discontinued user in the live fixture. |
| TC-009 | PASS | live: `Summary — anonymous` → 401, shared error envelope, no summary payload; `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/anonymous` | `E-1-CMD-AUT-001`. |
| TC-010 | PASS | live: `Summary — invalid token` → 401, same code; same handler test, `invalid token` sub-case | The token is absent from Redis, so it is a real cache miss. |
| TC-011 | PASS | live: user 1 `{5,1}` and user 3 `{0,0}` from their own tokens; `TestSummaryHandler_TC011_IsolatedPerUser` | Each token reaches the query as its own `user_id`; the counts never mix. |
| TC-012 | PASS | live: `?user_id=1&total=999&discontinued=999` on user 3's token → still `{0,0}`; `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored` | The query string is parsed and then discarded, so it cannot override identity or counts. |
| TC-013 | PASS | same live request (count overrides); same handler test also sends paging parameters | Paging means nothing on this route and changes nothing. |
| TC-014 | BLOCKED | `TestSummary_TC014_DerivedPerRequest` proves the service keeps no state across calls; `tests/db/TestSummary_TC014_ReflectsCatalogChangesAtRequestTime` skipped | Changing `is_available` between two requests needs a DB write the collection cannot perform. |
| TC-015 | PASS | `TestSummary_TC015_InternalFailureRenders500Envelope` — a DAO error produces 500, non-empty code/message, no counts | Not reachable live without breaking Postgres mid-run. |
| TC-016…TC-035 | BLOCKED | not implemented here | UI-level cases: `/fe-dev` scope in `sandbox-svc-ui`. `architecture.md` deliberately records no frontend PART ID, so these cannot appear as a `/go-dev` PART PR. |

## Live verification

**PASS.** `verify_stack` returned:

```
PASS (newman, 155s): stack healthy and every Newman request/assertion passed
app / postgres / postgres-seed / redis / redis-seed — all running, healthy
Newman exit code 0: 14 requests, 25 test-scripts, 64 assertions, 0 failed
```

The stack is the shipped `docker-compose.yml` with
`docker-compose.verify.yml` over it, seeded by `tests/live/postgres-seed.sql`
and `tests/live/redis-seed.sh`.

Summary route results from that run:

| Request | Result |
|---|---|
| `Summary — user 1 (5 total, 1 discontinued)` | 200, `application/json`, envelope integer-exact `{5,1}` |
| `Summary — user 3 (zero favorites)` | 200, `{0,0}` |
| `Summary — caller-supplied user id and counts are ignored` (`?user_id=1&total=999&discontinued=999`) | 200, still `{0,0}` on user 3's token |
| `Summary — anonymous` | 401, shared error envelope, no summary payload |
| `Summary — invalid token` | 401, same code as anonymous |
| `AppShell badge tooltip — summary shown on hover` | 200, `{5,1}` — the contract the tooltip consumes |

The FEAT-001 and FEAT-003 requests in the same collection also passed, so the
new route did not disturb the existing ones.

Earlier attempts in this session failed before any container started — the
runner reported "fetch failed", then a snapshot size/path problem from the
invocation path. Those were environment and invocation failures, not stack
failures, and are superseded by the PASS above.

## Observability

`architecture.md` decided to **reuse** the existing status-aware middleware; no
code was added for it. The route is registered on the same `*mux.Router`, and
the live scrape confirms it is recorded automatically under the literal
template `/v1/favorites/summary`.

The `Scrape metrics` request passed every assertion:

- `my_app_http_hist_count{method="GET",path="/v1/favorites/summary",status="200"}` present;
- a `status="401"` sample for the same path after the anonymous request;
- exactly one `200` sample — no duplicate label set;
- no query-string variant of the label;
- `/metrics` does not count itself;
- the same checks still hold for `/v1/favorites/export`.

## Deviations from the feature docs

1. **The success body is the nexcommon envelope.** `spec.md` writes the body as
   the bare object `{"total":…,"discontinued":…}`. The service returns
   `payload: {"total":…,"discontinued":…}` inside the shared
   `{success, header, payload}` envelope, like every other route in this
   service. This is architecture.md's recorded deviation 1, not a new one.
2. **The 401 code is nexcommon's `E-1-CMD-AUT-001`.** `spec.md` shows a
   placeholder `error_code`; the shared auth middleware supplies the real
   code. Also recorded by architecture.md.
3. **No frontend PART.** TC-016…TC-035 are `/fe-dev` work in
   `sandbox-svc-ui`; the tooltip itself is not in this repository.

No `dev_feedback` entry was added to `architecture.md`: nothing in the design
conflicted with what the codebase actually contains that the architecture had
not already recorded.

## Known limitations

- **`tests/db` did not execute.** `tests/testdb` starts Postgres through the
  Docker CLI; there is no Docker socket in this environment. The tests compile
  and are skipped with an explicit line, so the SQL cases above with no live
  counterpart (TC-003, TC-005…TC-008, TC-014) are BLOCKED rather than passed.
  Running `go test ./tests/db/...` on a machine with Docker (or in CI) executes
  them against the same statements the live route used.
- **No CI workflow exists in this repository.** `.github/workflows` is absent,
  so no CI-produced JUnit artifact was available or added. Adding CI is outside
  this feature's write scope; the JUnit run configured in
  `tests/live/run-go-tests.sh` / `docker-compose.test.yml` is the repo's
  existing mechanism.
- **`-race` did not run.** The toolchain here has no cgo/gcc, so `go test
  -race` refuses to build. The new production code has no goroutines; the only
  concurrency is the test-only fake Redis server, whose accept loop stops on
  listener close and whose pool reaper is now closed by `t.Cleanup`.
- **`verify_stack` needs an absolute `repo_path` here.** A relative path is
  resolved against the session root, not the working directory, so `"."`
  snapshots the whole session (over 100 MB) and `"work/code"` tripped the
  snapshot's path check. The absolute path passed.

## Delivery guard

| Check | How | Result |
|---|---|---|
| SQL / database security | `nexcraft:sql-security-validator`, gate mode, diff-scoped | **PASS** — the aggregate is a fixed literal with `$1` bound; no blockers, major or minor findings |
| Production readiness | `nexcraft:production-readiness-agent` is **not offered** in this session's agent list; substituted `feature-dev:code-reviewer` on the same diff plus a manual diff-scoped review | see the reviewer findings below |
| Vulnerability surface | manual: no new dependency, no secret, no new request input, parameterized query | **PASS** |
| Concurrency | manual: new code is one read-only `QueryRowContext`; no shared mutable state; `Add`'s transaction and lock untouched | **PASS** |
| Spec compliance | manual against `spec.md` REQ-001…REQ-029 | **PASS for the backend scope**; REQ-005…REQ-008, REQ-015…REQ-018, REQ-021, REQ-023, REQ-027…REQ-029 are UI behaviour owned by `/fe-dev` |
| Architecture compliance | manual against `architecture.md` PART-1 / PART-2 | **PASS** — same files, same signatures, the reuse-only Observability decision honoured |
| Bundle coverage | `verify_bundle_coverage.py` | **PASS** — 26 registered codes, all bundled |
| Generated layout | `verify_generated_go_layout.py` | **PASS** — no misplaced files, no template artifacts |
| Go build / vet / format | `go build`, `go vet`, `gofmt -l` | **PASS** |
| Go tests | `go test ./...` | **PASS** — every package, `tests/db` skipped |
| Live stack | `verify_stack` — compose up, Newman, teardown | **PASS** (newman, 155s) — 14 requests, 64 assertions, 0 failed |

### Review findings and what happened to them

The substituted reviewer raised three findings; the two defects were fixed
before this file was finalized, the third was judged cosmetic and left alone.

1. **Leak — the fake Redis client was never closed.** `redis.NewClient` starts
   the pool's idle reaper and only `Close` stops it, so six clients leaked per
   test binary. Fixed: `fakeRedis` now registers `t.Cleanup(client.Close)`.
2. **Vacuous assertions — the 401 and 500 "no counts" checks read the top-level
   object instead of `payload`.** Since the envelope is `{success, header,
   payload}`, leaked counts would never have appeared at the top level, so the
   assertions could not fail. Fixed: both now decode `payload` and check
   `total` / `discontinued` there.
3. **Cosmetic, not applied — `registerSummary` omits `NotReadBody()`.** The
   generator therefore emits an empty `requestBody` in its own `/inspect`
   document. This matches `registerList`, and `api/openapi.yaml` (the shipped
   artifact) declares no request body, so the runtime contract is correct.

The reviewer otherwise confirmed the aggregate, the service's identity source,
the DTO tags, the route-ordering behaviour and the absence of shared mutable
state or goroutines in the new read path.
