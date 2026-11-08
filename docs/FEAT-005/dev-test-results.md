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
to run Docker in this environment. Its parameters, as used for the attempt:

```
repo_path: work/code
compose_file: docker-compose.yml
compose_override: docker-compose.verify.yml
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

`live` would mean the assertion ran against the real stack through Newman; it
did not run in this session — every live attempt failed in the runner
environment (see Live verification). Any other evidence named below is a Go
test that executed here, except those explicitly marked *skipped*.

The handler tests drive the real registered route, the real
`UserAccessValidator` → Redis → context path (against an in-process RESP2
server), the real DTO serialisation and the real error formator. The DAO is a
fake, so they prove the HTTP contract and the identity/derivation rules but not
the SQL.

| Case ID | Result | Evidence | Notes |
|---|---|---|---|
| TC-001 | PASS | `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued` — 200, `application/json`, payload exactly `{3,1}`; `TestFavoriteSummaryMarshalsExactIntegers` | The aggregate SQL itself did not execute here; `tests/db/TestSummary_TC001_CountsTotalAndDiscontinued` covers it but is skipped. |
| TC-002 | PASS | `TestSummaryHandler_TC002_ZeroFavoritesIsA200` — 200 with integer `{0,0}`, not 404/204/empty | |
| TC-003 | BLOCKED | `TestSummary_TC003_SingleAvailableFavorite` (service mapping) passes; `tests/db/TestSummary_TC003_SingleAvailableFavorite` is **skipped** | No Docker socket; live runner unavailable. The SQL boundary was not executed. |
| TC-004 | BLOCKED | same shape: service test passes, `tests/db/TestSummary_TC004_MixedPartition` skipped | The `COUNT(*) FILTER` partition is only executable against Postgres. |
| TC-005 | BLOCKED | service test passes; `tests/db/TestSummary_TC005_AllDiscontinuedEqual` skipped | |
| TC-006 | BLOCKED | service test passes; `tests/db/TestSummary_TC006_FiveHundredCap` skipped | |
| TC-007 | BLOCKED | service test passes; `tests/db/TestSummary_TC007_AboveNinetyNineNotCapped` skipped | |
| TC-008 | BLOCKED | service test passes; `tests/db/TestSummary_TC008_SingleDiscontinued` skipped | |
| TC-009 | PASS | `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/anonymous` — 401, `E-1-CMD-AUT-001`, no `total`/`discontinued` | |
| TC-010 | PASS | same test, `invalid token` sub-case — identical envelope | |
| TC-011 | PASS | `TestSummaryHandler_TC011_IsolatedPerUser` — two tokens, each reaches the DAO with its own user id and gets its own payload | SQL scoping (`WHERE user_id = $1`) is covered by `tests/db/TestSummary_TC011_IsolatedPerUser`, skipped here. |
| TC-012 | PASS | `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored` — `?user_id=2&total=999&discontinued=999` plus `X-User-Id: 2` change nothing; DAO received the token's user | |
| TC-013 | PASS | same test — paging and count overrides ignored | |
| TC-014 | BLOCKED | `TestSummary_TC014_DerivedPerRequest` proves the service keeps no state across calls; `tests/db/TestSummary_TC014_ReflectsCatalogChangesAtRequestTime` skipped | The live catalog re-read is SQL, so the case's defining step did not execute. |
| TC-015 | PASS | `TestSummary_TC015_InternalFailureRenders500Envelope` — a DAO error produces 500, non-empty code/message, no counts | |
| TC-016…TC-035 | BLOCKED | not implemented here | UI-level cases: `/fe-dev` scope in `sandbox-svc-ui`. `architecture.md` deliberately records no frontend PART ID, so these cannot appear as a `/go-dev` PART PR. |

## Live verification

**Not run — the verification runner was unavailable.** `verify_stack` never
returned PASS, so no live result is claimed. The attempts with the correct
repository path failed with:

```
The verification runner is unavailable: fetch failed.
```

(Earlier attempts from the wrong working directory failed with "snapshot
exceeds 100 MB" and a `tar` path error; those were invocation-path problems,
not stack failures, and are not counted as results.)

Scope of the failure: the runner could not be reached at all, so no service
started, no Newman request ran, and no `/metrics` scrape occurred. This is an
environment failure, not a failed test. The stack, the fixture and the
collection are in place and are the exact inputs listed above; re-running them
on a working runner is the remaining step.

What the run would assert, already encoded in `postman/collection.json`:

| Check | Case | Assertion |
|---|---|---|
| User 1's totals | TC-001, TC-011 | 200, payload exactly `{5,1}` from the seeded five favorites (item 102 discontinued) |
| User 3's totals | TC-002, TC-011 | 200, payload exactly `{0,0}` |
| Caller-supplied input | TC-012, TC-013 | `?user_id=1&total=999&discontinued=999` and `X-User-Id: 1` leave user 3's answer at `{0,0}` |
| Anonymous and invalid token | TC-009, TC-010 | 401, `E-1-CMD-AUT-001`, no counts |
| FE surface | TC-016 (contract half) | the AppShell badge-tooltip folder receives `{5,1}` |

The Go tests cover the whole HTTP contract in the meantime, including the 200,
0, 401 and 500 paths.

## Observability

`architecture.md` decided to **reuse** the existing status-aware middleware; no
code was added for it. The route was registered on the same `*mux.Router`, so
the middleware records it automatically under the literal template
`/v1/favorites/summary`.

Live verification **did not run** (runner unavailable), so the scrape evidence
is not claimed. The checks are encoded in the `Scrape metrics` request and will
run unchanged:

- one `my_app_http_hist_count{method="GET",path="/v1/favorites/summary",status="200"}` sample;
- a `status="401"` sample for the same path after the anonymous request;
- exactly one `200` sample — no duplicate label set;
- no query-string variant of the label;
- no `path="/metrics"` sample.

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
  and are skipped with an explicit line, so every SQL-level case above is
  BLOCKED rather than passed. Running `go test ./tests/db/...` on a machine
  with Docker (or in CI) executes them.
- **No CI workflow exists in this repository.** `.github/workflows` is absent,
  so no CI-produced JUnit artifact was available or added. Adding CI is outside
  this feature's write scope; the JUnit run configured in
  `tests/live/run-go-tests.sh` / `docker-compose.test.yml` is the repo's
  existing mechanism.
- **`-race` did not run.** The toolchain here has no cgo/gcc, so `go test
  -race` refuses to build. The new production code has no goroutines; the only
  concurrency is the test-only fake Redis server, whose goroutines terminate
  when the listener is closed and which the tests exercise serially.
- **The live stack and its metrics scrape are unverified in this session** for
  the runner reason above.

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
