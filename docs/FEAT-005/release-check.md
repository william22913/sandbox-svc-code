# FEAT-005 — Release Check (sandbox-svc)

| | |
|---|---|
| Service | `sandbox-svc` |
| Feature | `FEAT-005` — Favorites summary and badge tooltip |
| Spec version | `1` — `docs/FEAT-005/spec.md` |
| Testcase version | `1` — `docs/FEAT-005/testcase.md` |
| Architecture version | `1` — `docs/FEAT-005/architecture.md` |
| Database version | `1` — `docs/FEAT-005/database.md` |
| code_repo | https://github.com/william22913/sandbox-svc-code |
| release_branch (from architecture.md) | `main` |
| integration_branch (from architecture.md) | `feat/FEAT-005` |
| Tested integration commit | `ed2095af53ae252fe7531cded4142449ac58eb0a` |
| Part PRs | `PART-1` → https://github.com/william22913/sandbox-svc-code/pull/3 ; `PART-2` → https://github.com/william22913/sandbox-svc-code/pull/3 |
| OpenAPI source | `api/openapi.yaml` — SHA-256 `c4bf73b05c8478c4fd476e6ad34667b10b88b80ba11379c55961cfe9ce0b14e8` |
| Postman collection source | `postman/collection.json` — SHA-256 `57cf8c2b288acab3bd0993ca62e176ed839624ea63d38969d4a2685a1fb6e874` |
| Postman local environment source | `postman/environment.local.json` — SHA-256 `c441473859c891987e003912e0c93c52fe5c09b796c54bd0b10d108a2d69f32b` |
| Code release CI | https://github.com/william22913/sandbox-svc-code/actions/runs/42457539537 — `success` |
| Release PR | pending |

## Input verification

- The four approved handoff copies (`spec.md`, `testcase.md`,
  `architecture.md`, `database.md`) on `feat/FEAT-005` are byte-identical to
  the docs repo base branch (`main`) copies under
  `sandbox-svc/feature/FEAT-005/` — checked with `cmp`.
- The code repo origin (`https://github.com/william22913/sandbox-svc-code`)
  matches `code_repo` in `architecture.md:7`.
- `release_branch` `main` (`d7fd70bc6f5e333f08206eab400b9e3ff56a5864`) is an
  ancestor of `Tested integration commit` (`git merge-base --is-ancestor`), so
  the CI run below reflects `release_branch`'s current state.
- `docs/FEAT-005/dev-test-results.md` exists on the integration branch. Its
  `Part PRs:` line names PART-1 and PART-2 → PR #3, which the git host reports
  merged (`merged: true`, base `feat/FEAT-005`) with title
  `[DEV] sandbox-svc/FEAT-005: favorites summary endpoint (PART-1..PART-2)`.

## Implementation Plan coverage

Source: `dev-test-results.md` `Part PRs:` line, cross-checked against the git
host PR record for #3.

| PART ID | Dev PR | Status |
|---|---|---|
| PART-1 | https://github.com/william22913/sandbox-svc-code/pull/3 | MERGED |
| PART-2 | https://github.com/william22913/sandbox-svc-code/pull/3 | MERGED |

Both PART IDs are delivered by the same `/go-dev` PR, whose title names the
range `PART-1..PART-2`; `architecture.md` records no frontend PART ID by
design (`docs/FEAT-005/architecture.md:578`).

## Feature requirements

`Evidence` cites the CI run's JUnit report
(https://github.com/william22913/sandbox-svc-code/actions/runs/42457539537,
artifact `junit-report`, 97 tests / 0 failures) by test name, or a
`path:line` where the behaviour is implemented. UI-only requirements are owned
by `/fe-dev` per `testcase.md`'s `- **Level**:` rule.

| Spec reference | Implementation | Evidence | Status |
|---|---|---|---|
| REQ-001 | `GET /v1/favorites/summary` → service → DAO aggregate | `services/favorite/summary_favorite.go:19`; `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued` | PASS |
| REQ-002 | Standard success envelope with integer `payload.total`/`payload.discontinued`; `200`; `application/json` | `dto/out/favorite.go:44`; `TestFavoriteSummaryMarshalsExactIntegers`; `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued` | PASS |
| REQ-003 | `COUNT(*)` with no availability predicate | `dao/favorite_dao.go:208`; `TestSummary_TC001_CountsTotalAndDiscontinued` | PASS |
| REQ-004 | `COUNT(*) FILTER (WHERE NOT COALESCE(c.is_available, FALSE))` | `dao/favorite_dao.go:215`; `TestSummary_TC004_MixedPartition`; `TestSummary_TC005_AllDiscontinuedEqual` | PASS |
| REQ-005 | Tooltip literal `N favorites (M discontinued)`, exact integers | `docs/FEAT-005/testcase.md:223` (TC-016 `Level: UI`) | FE-owned |
| REQ-006 | Tooltip only for a visible FEAT-004 badge | `docs/FEAT-005/testcase.md:287` (TC-023 `Level: UI`) | FE-owned |
| REQ-007 | First-hover fetch, refresh after favorite action | `docs/FEAT-005/testcase.md:227` (TC-017 `Level: UI`) | FE-owned |
| REQ-008 | No tooltip while loading/failed; retry on later hover | `docs/FEAT-005/testcase.md:327` (TC-027 `Level: UI`) | FE-owned |
| REQ-009 | Non-negative integers derived per request, `0 <= discontinued <= total` | `dao/favorite_dao.go:208`; `TestSummary_TC014_DerivedPerRequest`; `TestSummary_TC006_FiveHundredCap` | PASS |
| REQ-010 | User id read only from auth context | `services/favorite/summary_favorite.go:25`; `TestSummaryHandler_TC011_IsolatedPerUser` | PASS |
| REQ-011 | No request input can influence counts | `endpoint/favorite_endpoint/favorite_endpoint.go:111`; `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored` | PASS |
| REQ-012 | Zero favorites → `200` `{0,0}` | `TestSummaryHandler_TC002_ZeroFavoritesIsA200`; `TestSummary_TC002_ZeroFavorites` | PASS |
| REQ-013 | Anonymous → `401` standard error envelope, no summary | `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/anonymous` | PASS |
| REQ-014 | All discontinued → `total == discontinued` | `TestSummary_TC005_AllDiscontinuedEqual`; `TestSummary_TC008_SingleDiscontinued` | PASS |
| REQ-015 | Hidden badge → no tooltip, no fetch | `docs/FEAT-005/testcase.md:293` (TC-024 `Level: UI`) | FE-owned |
| REQ-016 | Loading/failure → no spinner/error UI | `docs/FEAT-005/testcase.md:327` (TC-027 `Level: UI`) | FE-owned |
| REQ-017 | Backend failure → `500` standard envelope, no partial counts; FE shows no tooltip | `TestSummary_TC015_InternalFailureRenders500Envelope`; FE half `docs/FEAT-005/testcase.md:337` (TC-028 `Level: UI`) | PASS (backend); FE half FE-owned |
| REQ-018 | Pre-action summary never re-shown | `docs/FEAT-005/testcase.md:347` (TC-029 `Level: UI`) | FE-owned |
| REQ-019 | Reads existing `favorite_items`/`catalog_items`; no schema change | `docs/FEAT-005/database.md:93`; `sql_migrations/main/` unchanged (3 files, none added) | PASS |
| REQ-020 | Row ownership via auth-context scope | `dao/favorite_dao.go:219`; `TestSummary_TC011_IsolatedPerUser` | PASS |
| REQ-021 | Extends FEAT-004 badge, no second badge/provider | `docs/FEAT-005/testcase.md:267` (TC-021 `Level: UI`) | FE-owned |
| REQ-022 | Reuses shared `UserAccessValidator()`; no new login logic | `endpoint/favorite_endpoint/favorite_endpoint.go:119`; `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/invalid_token` | PASS |
| REQ-023 | Same post-action refresh trigger | `docs/FEAT-005/testcase.md:347` (TC-029 `Level: UI`) | FE-owned |
| REQ-024 | `200` with `total = T`, `discontinued = D` for `0 <= D <= T` | `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued`; `TestSummary_TC001_CountsTotalAndDiscontinued` | PASS |
| REQ-025 | Unauthenticated → `401`, no summary body | `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/anonymous` | PASS |
| REQ-026 | No request input can include another user's favorites | `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored`; `TestSummary_TC013_ExtraQueryParametersCannotOverride` | PASS |
| REQ-027 | Visible-badge tooltip exact text under `99+`; hidden → none | `docs/FEAT-005/testcase.md:257` (TC-020 `Level: UI`) | FE-owned |
| REQ-028 | Loading/failed/`500` → no tooltip, badge unchanged | `docs/FEAT-005/testcase.md:327` (TC-027 `Level: UI`) | FE-owned |
| REQ-029 | Post-action tooltip reflects new values | `docs/FEAT-005/testcase.md:347` (TC-029 `Level: UI`) | FE-owned |
| Non-goal: no click-through | No click handler / no navigation surface | `docs/FEAT-005/testcase.md:387` (TC-033 `Level: UI`) | FE-owned |
| Non-goal: no polling/realtime/cross-tab sync | No such behaviour implemented in this pass | `docs/FEAT-005/testcase.md:52` (marked Deferred) | Deferred |
| Non-goal: no server-side or persisted client cache | Values computed per request; nothing stored | `TestSummary_TC014_ReflectsCatalogChangesAtRequestTime` | PASS (backend); FE state FE-owned |
| Non-goal: no per-item breakdown | Response carries only `total`/`discontinued` | `dto/out/favorite.go:44` | PASS |
| Non-goal: no admin/cross-user summary | Per-token scope only | `TestSummaryHandler_TC011_IsolatedPerUser` | PASS |
| Non-goal: no change to FEAT-004's badge | No badge code in this repo; FE owns it | `docs/FEAT-005/testcase.md:267` (TC-021 `Level: UI`) | FE-owned |
| Non-goal: no new page/settings/notification/history | No such surface added | `docs/FEAT-005/architecture.md:560` | Not applicable |
| Non-goal: no tooltip trigger other than hover | Hover-only; no focus/click | `docs/FEAT-005/testcase.md:387` (TC-033 `Level: UI`) | FE-owned |
| Non-goal: no localization/pluralization | Literal template required by spec | `docs/FEAT-005/testcase.md:247` (TC-019 `Level: UI`) | FE-owned |
| Non-goal: no tooltip animation | No animation requirement, none added | `docs/FEAT-005/architecture.md:560` | Not applicable |
| Non-goal: no analytics/post-launch metric | None added | `docs/FEAT-005/architecture.md:586` | Not applicable |
| Non-goal: no new table/column/index/cache | Zero migration batches; no `sql_migrations` change | `docs/FEAT-005/database.md:95` | PASS |

## Test cases

The CI JUnit report (run 42457539537, `Tested integration commit`
`ed2095af…`) is the evidence for backend-scope cases — not the per-session
rows in `dev-test-results.md`, which tested each part in isolation. TC-016…TC-035
are `Level: UI` and are executed by `/fe-dev` in `sandbox-svc-ui`; they do not
block this backend release.

| Case ID | Result | Evidence | Status |
|---|---|---|---|
| TC-001 | PASS | `TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued`; `TestSummary_TC001_CountsTotalAndDiscontinued` | covered |
| TC-002 | PASS | `TestSummaryHandler_TC002_ZeroFavoritesIsA200`; `TestSummary_TC002_ZeroFavorites` | covered |
| TC-003 | PASS | `TestSummary_TC003_SingleAvailableFavorite` (service + db) | covered |
| TC-004 | PASS | `TestSummary_TC004_MixedPartition` (service + db) | covered |
| TC-005 | PASS | `TestSummary_TC005_AllDiscontinuedEqual` (service + db) | covered |
| TC-006 | PASS | `TestSummary_TC006_FiveHundredCap` (service + db) | covered |
| TC-007 | PASS | `TestSummary_TC007_AboveNinetyNineNotCapped` (service + db) | covered |
| TC-008 | PASS | `TestSummary_TC008_SingleDiscontinued` (service + db) | covered |
| TC-009 | PASS | `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/anonymous` | covered |
| TC-010 | PASS | `TestSummaryHandler_TC009_TC010_RejectsUnauthenticated/invalid_token` | covered |
| TC-011 | PASS | `TestSummaryHandler_TC011_IsolatedPerUser`; `TestSummary_TC011_IsolatedPerUser` | covered |
| TC-012 | PASS | `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored`; `TestSummary_TC012_CallerInputIsIgnored` | covered |
| TC-013 | PASS | `TestSummaryHandler_TC012_TC013_RequestInputIsIgnored`; `TestSummary_TC013_ExtraQueryParametersCannotOverride` | covered |
| TC-014 | PASS | `TestSummary_TC014_DerivedPerRequest`; `TestSummary_TC014_ReflectsCatalogChangesAtRequestTime` | covered |
| TC-015 | PASS | `TestSummary_TC015_InternalFailureRenders500Envelope`; `TestSummary_TC015_UnexpectedFailurePassesThrough` | covered |
| TC-016 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:223` (Level: UI) | FE scope |
| TC-017 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:233` (Level: UI) | FE scope |
| TC-018 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:243` (Level: UI) | FE scope |
| TC-019 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:253` (Level: UI) | FE scope |
| TC-020 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:263` (Level: UI) | FE scope |
| TC-021 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:273` (Level: UI) | FE scope |
| TC-022 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:283` (Level: UI) | FE scope |
| TC-023 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:293` (Level: UI) | FE scope |
| TC-024 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:303` (Level: UI) | FE scope |
| TC-025 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:313` (Level: UI) | FE scope |
| TC-026 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:323` (Level: UI) | FE scope |
| TC-027 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:333` (Level: UI) | FE scope |
| TC-028 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:343` (Level: UI) | FE scope |
| TC-029 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:353` (Level: UI) | FE scope |
| TC-030 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:363` (Level: UI) | FE scope |
| TC-031 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:373` (Level: UI) | FE scope |
| TC-032 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:383` (Level: UI) | FE scope |
| TC-033 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:393` (Level: UI) | FE scope |
| TC-034 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:403` (Level: UI) | FE scope |
| TC-035 | Owned by /fe-dev | `docs/FEAT-005/testcase.md:413` (Level: UI) | FE scope |

## Logs

| Check | Evidence | Status |
|---|---|---|
| Error lifecycle log on aggregate failure | `dao/favorite_dao.go:223` logs the DAO error with a static message and returns it for the standard `500` — no values are rendered from it | PASS |
| No secret or PII logging | `dao/favorite_dao.go:223` logs only `err`; no token, no user identifier, no favorite contents. The summary service (`services/favorite/summary_favorite.go:19`) adds no logging | PASS |
| Correlation | The response header carries `request_id` in the standard envelope (`api/openapi.yaml:421`) | PASS |
| Feature-specific audit/lifecycle events | None required by `spec.md` or `architecture.md` — this is a read-only aggregate with no state transition to audit | Not applicable |

## Metrics

`architecture.md` (`## Observability`, line 586) decided to **reuse** the
existing status-aware middleware with no feature-specific metric.

| Check | Evidence | Status |
|---|---|---|
| Route observed under its own literal template | `middleware/metrics.go:81`; Newman `Scrape metrics` assertion "the summary route is observed under its own path template" | PASS |
| Success and error statuses are distinguished | `middleware/metrics.go:67`; `TestScrapeSeparatesSuccessFromCapRejection`; Newman "a rejected summary is observed as 401, not 200" | PASS |
| No duplicate/relabelled samples; no query-string labels | `TestScrapeLabelsByRouteTemplateNotByID`; Newman "exactly one summary 200 sample — no duplicate labels" and "the summary route is not relabelled with a query string" | PASS |
| Business metric | None required (`spec.md` Success Criteria, line 232) | Not applicable |

## Deployment and contract

Named environment for the live run: the compose stack from
`docker-compose.yml` + `docker-compose.verify.yml`, service `app` on container
port `8080`, Postgres and Redis wired by service name.

| Check | Evidence | Status |
|---|---|---|
| CI on `Tested integration commit` | https://github.com/william22913/sandbox-svc-code/actions/runs/42457539537 — `success`, 97 tests, 0 failures | PASS |
| Docker build | `Dockerfile:1` (vendored Go 1.27 build) — `verify_stack` built and started the image | PASS |
| Compose up / required dependencies | `docker-compose.yml:1` (app + postgres + redis, health-gated) — `verify_stack` PASS: all five services healthy | PASS |
| Migrations plan/order/rollback | `docs/FEAT-005/database.md:95` — zero batches; `sql_migrations/main/1..3` unchanged, no rollback needed | PASS |
| Env/config | `docker-compose.yml:39` (`SANDBOX_SVC_POSTGRESQL_HOST`/`SANDBOX_SVC_REDIS_HOST`); published `18080:8080` at `docker-compose.yml:44` | PASS |
| Generated OpenAPI path | `api/openapi.yaml:126` (`/v1/favorites/summary`), schema at `api/openapi.yaml:394`, envelope at `api/openapi.yaml:456` | PASS |
| Live API/Newman | `verify_stack` PASS (newman, 20s): 14 requests, 25 test-scripts, 64 assertions, 0 failed; `Summary — user 1` → 200 `{5,1}`, `Summary — user 3` → 200 `{0,0}`, anonymous/invalid → 401 | PASS |
| Contract drift vs `architecture.md` `## Backend — Endpoints` | `architecture.md:402` declares one route, `GET /v1/favorites/summary`, consumer `fe` — matches `api/openapi.yaml:126` exactly (path, method, `payload.total`/`payload.discontinued` integers, 200/401/500). No other endpoint row exists | PASS |
| Breaking changes vs docs base branch | No prior `openapi.yaml` for this service exists in the docs repo (`find` returned none), so this is the first published contract; `architecture.md:569` records no planned breaking changes | PASS (no baseline) |
| `TestOpenAPISpecMatchesRegisteredRoutes` | CI suite asserts the shipped spec matches the registered routes | PASS |

## Open issues

None blocking.

Non-blocking observations:

1. `docs/FEAT-005/dev-test-results.md` is stale at the current tip: it says
   "No CI workflow exists" (`:166`) and marks TC-003/005/006/007/008/014
   `BLOCKED` (`:74`–`:85`) because `tests/db` could not run without Docker.
   Commits `c17389f`/`ed2095a` added CI, and the run above executed those DB
   tests with 0 failures. The file is read-only evidence for this command;
   `/go-dev` may refresh it if desired — it does not affect this release.
2. TC-016…TC-035 remain `/fe-dev` work in `sandbox-svc-ui`; the tooltip
   feature is not released by this backend PR.

Release ready: yes
