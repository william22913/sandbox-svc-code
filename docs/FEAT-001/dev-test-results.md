# FEAT-001 — Dev Test Results

Written by `/go-dev` for `sandbox-svc` / `FEAT-001`. One line per test case
with its evidence, then the cases that cannot be settled at the backend layer,
then the deliberate deviations and the known limitations of this pass.

## How to run

```sh
go build -mod=vendor ./...
go test -mod=vendor -count=1 ./tests/...
```

`tests/db` starts its own Postgres container and exits 0 with a printed SKIP
when Docker is unavailable, so the suite is not silently green on a machine
that never ran those tests — read the output.

## Implementation points

| # | Point | Files | Tests |
|---|---|---|---|
| 1 | Schema + migrations (`favorite_items`, `users`, `catalog_items`) | `sql_migrations/main/*.sql` | `tests/db` (they run against it) |
| 2 | Sentinels + bundles | `error/error.go`, `i18n/common/error/en-US.json`, `constanta/` | `tests/bundle_coverage_test.go` |
| 3 | DAO | `dao/favorite_dao.go` | `tests/db` |
| 4 | Input DTO + validation | `dto/in/favorite.go` | `tests/dto` |
| 5 | Output DTOs | `dto/out/favorite.go` | `tests/dto` |
| 6 | Service | `services/favorite/*.go` | `tests/service` |
| 7 | Endpoint registration | `endpoint/favorite_endpoint/`, `router/path.go`, `server_config/server_{dao,services,endpoint}.go` | `tests/endpoint` |
| 8 | Observability | `middleware/metrics.go`, `router/controller.go` | `tests/middleware` |

## Test case results

Verdicts are for the **backend layer only**. Every case in `testcase.md` except
TC-015 and TC-016 is written from the UI's point of view ("tap the star
icon"), so several have a UI half this pass cannot verify; those are marked
below and listed again under Test case feedback.

| TC | Requirement | Verdict | Evidence |
|---|---|---|---|
| TC-001 | REQ-001 favorite an item | PASS (backend half) | `TestAddInserted`, `TestAddInsertsThenIsIdempotent`, `TestRoutesAreRegisteredWithTheirMethods`. The star's rendered state is the UI's. |
| TC-002 | REQ-001 appears immediately, no reload | PASS (backend half) | Backend supplies the assertion: the add response is authoritative and the next `List` returns the item — `TestAddInserted`, `TestListMapsRowsAndKeepsAnEmptyListAnArray`. "Without reload" is a UI property. |
| TC-003 | REQ-002 lists all favorited items | PASS | `TestListMapsRowsAndKeepsAnEmptyListAnArray`, `TestListReportsDiscontinuedItemAsUnavailable` (2 distinct items, no duplication or omission) |
| TC-004 | REQ-002 empty list, no error | PASS | `TestFavoriteListEmptyIsArrayNotNull` (asserts the body is exactly `{"favorites":[]}`), `TestListIsEmptyNotNilWithoutFavorites`, `TestListMapsRowsAndKeepsAnEmptyListAnArray` |
| TC-005 | REQ-003 un-favorite | PASS (backend half) | `TestRemoveDeletesScopedToTheAuthenticatedUser`, `TestRemoveOnlyTouchesTheCallersOwnRow` |
| TC-006 | REQ-003 un-favorite from list, immediate | PASS (backend half) | Same as TC-005. "Immediately" is a UI property. |
| TC-007 | REQ-004 discontinued item stays, marked unavailable | PASS (backend half) | `TestListReportsDiscontinuedItemAsUnavailable` — the item is favorited while available, discontinued by a soft mark, and still returned with `available: false` and its catalog name. The "unavailable" badge is the UI's. |
| TC-008 | REQ-004 un-favorite a discontinued item | PASS | `TestRemovingADiscontinuedItemStillWorks` |
| TC-009 | REQ-005 re-favoriting is idempotent | PASS | `TestAddAlreadyFavoritedIsSuccess`, `TestAddInsertsThenIsIdempotent` (asserts exactly one row), `TestRefavoritingAtCapStillSucceeds` |
| TC-010 | REQ-006 anonymous cannot favorite | OUT OF BACKEND SCOPE | See Test case feedback. |
| TC-011 | REQ-006 redirect target is login | OUT OF BACKEND SCOPE | See Test case feedback. |
| TC-012 | REQ-007 the 500th favorite is accepted | PASS | `TestAddAtCapFailsWithoutInsertingOrEvicting` (seeds `limit` favorites at a small limit, asserts the boundary inserts), `TestAddDelegatesTheAuthenticatedUserAndTheCap` (asserts the production cap is 500) |
| TC-013 | REQ-007 the 501st returns a clear error | PASS | `TestAddAtCapFailsWithoutInsertingOrEvicting` — asserts the outcome is `AddLimitReached`, the count did not move, and the rejected item is not in the table. `TestAddAtCapReturnsLimitReached` maps it to a 422 `FAVORITES_LIMIT_REACHED`, not a 500. |
| TC-014 | REQ-007 un-favorite frees a slot | PASS | `TestUnfavoritingAtCapFreesASlot` |
| TC-015 | REQ-008 never-favorited returns 404 | PASS | `TestRemoveMissingRowReturnsNotFound` (→ `ErrFavoriteNotFound`), `TestRemoveNonNumericPathIsNotFound` |
| TC-016 | REQ-008 another user's item returns the same 404 | PASS | `TestRemoveOnlyTouchesTheCallersOwnRow` — user 2's delete leaves user 1's identical `(item_id)` row intact, and a cross-user remove is indistinguishable from a missing row. Only `ErrFavoriteNotFound` (404) exists on this path; there is no 403 sentinel to return. |

## Test case feedback

Two cases cannot be verified at this layer. `architecture.md`'s REQ-006 row
already says so — *"Enforced by the assumed shared auth middleware (not this
feature's code) … the FE's redirect-on-401 behavior is explicitly out of
scope this pass"* — so this is the docs' own position, recorded here rather
than passed silently.

- **TC-010** — "anonymous user cannot favorite". The 401 is produced by
  nexcommon's `UserAccessValidator()` before any feature code runs, and its
  Redis/JWT internals belong to the auth service. The backend-observable half
  of this case *is* covered: `TestDeleteIsDocumentedTaggedAndSecured` asserts
  the routes carry a security requirement, so they are not documented as
  publicly callable. Needs a real unauthenticated request against the running
  stack — Part 6's Newman run — to close it properly.
- **TC-011** — "redirect target is the login page". Purely a frontend
  behavior; there is no backend response shape to assert. The backend half,
  "no favorite record is created", is covered by TC-010's route protection.

Neither case is wrong, and neither is skipped: both are correctly written,
they just live a layer above this pass.

## Deviations from the feature docs

**Route prefix.** `architecture.md` writes the paths unprefixed
(`POST /favorites`); the implementation registers them under
`/v1/favorites`. The `/v1` comes from `router.V1PrefixPath`, which is where
the two reference nexcommon services on disk keep their API version, and it
matches this service's existing `/v1/health`. The doc's spelling reads as
shorthand for the resource rather than a statement about versioning. The
generated OpenAPI document, not `architecture.md`, is what a frontend pass
will build against.

**Error envelope.** `architecture.md` describes the cap error as
`{"error_code": "FAVORITES_LIMIT_REACHED", "message": "..."}`. The real
envelope is nexcommon's
`{"success": false, "header": {...}, "payload": {"status": 422, "code": "FAVORITES_LIMIT_REACHED", "message": "..."}}`.
The code string matches; the field names are the library's and this service
cannot change them. A frontend reading `error_code` would get `undefined` —
worth correcting in the doc before the FE pass.

**Plain `WrapService` for the list.** `GET /v1/favorites` is registered as a
plain service, not a get-list. `WrapServiceListData` mandates
`page`/`limit`/`filter`/`order`/`connector` query parameters and a
`[]model.SearchParam`; architecture.md defers sorting and filtering and says
the list is unordered this pass, so registering it as a get-list would
promise a paginated, filterable endpoint that does not exist.

**Non-positive `item_id` on DELETE returns 404, not 400.** A path segment
that is not a positive integer takes the same 404 as a well-formed id
matching no row. REQ-008 requires one indistinguishable answer for everything
that does not name one of the caller's own favorites, and a distinct 400
would be the only way this endpoint could answer differently. Documented in
`services/favorite/remove_favorite.go`.

## Known limitations

**`-race` was not available.** `go test -race` fails on this machine with
`cgo: C compiler "gcc" not found`, and there is no other C compiler on
`PATH`. The cap's concurrency guarantee — `TestAddCapHoldsUnderConcurrentRequests`,
which fires 8 concurrent adds for 1 free slot — therefore ran **without** the
race detector. It passes and asserts the invariant it can see (exactly one
insert, seven rejections, final count equal to the limit), but a data race
inside the DAO's transaction would not have been reported. CI should run that
test with `-race`.

**The swagger document drops `POST /v1/favorites`.** nexcommon's
`NewHandleFuncParam` assigns a fresh method map for a path on every call
(`SwaggerPath[path] = make(...)`), so two routes sharing a path overwrite each
other and only the last registered survives. The route is registered, routed
and tagged correctly; it is the generated document that loses it. Pinned by
`TestCollectionPathDocumentsOnlyOneMethod`, which fails once the library is
fixed. The OpenAPI artifact is hand-corrected in Part 6.

**404s are not measured.** gorilla/mux applies middleware only after a route
matches, so an unmatched request never reaches the metrics middleware and
produces no metric at all. The request histogram measures handled traffic and
cannot be used to alert on 404 volume. Pinned by
`TestUnmatchedRequestsAreNotRecorded`.

**Registration requires the validators to be wired first.** `WrapService`
builds each route's swagger entry immediately and dereferences the tag
validator while reading DTO fields, so calling `RegisterEndpoint` before
`TagValidator(...)` panics at startup rather than at request time. The
composition root orders this correctly — validators in `Init()`, routes in
`InitEndpoint()` — and `tests/endpoint` mirrors that order deliberately.

## Mutation checks

Each guard was verified to fail when the behavior it claims is broken, then
reverted.

| Mutation | Test that caught it |
|---|---|
| Drop `min:"1"` from `AddFavoriteRequest.ItemID` | `TestAddRequestRejectsNonPositiveItemID` |
| Drop `CATALOG_ITEM_NOT_FOUND` from the error bundle | `TestEveryErrorCodeIsBundled` |
| Change the cap from 500 to 100 | `TestAddDelegatesTheAuthenticatedUserAndTheCap` |
| Report the cap failure as success | `TestAddAtCapReturnsLimitReached` |
| Drop the DELETE route registration | `TestRoutesAreRegisteredWithTheirMethods`, `TestDeleteIsDocumentedTaggedAndSecured` |
| Tag the DELETE route with something other than `Favorite` | `TestDeleteIsDocumentedTaggedAndSecured` |
| Hardcode the metric status to 200 (nexcommon's own bug) | `TestScrapeSeparatesSuccessFromCapRejection` |
| Label metrics by `r.URL.Path` instead of the route template | `TestRequestsWithNoMatchedRouteUseTheFixedLabel` |
| Drop the fixed fallback metric label | `TestRequestsWithNoMatchedRouteUseTheFixedLabel` |

## Delivery guard (Part 5)

Run against this feature's diff only (`git diff origin/feat/FEAT-001...HEAD`,
excluding `vendor/`): 38 files, ~4.9k lines. Commands: `go vet`, `go build
-mod=vendor`, `go test -mod=vendor -count=1 ./tests/...` — all clean; six test
packages pass.

### SQL security gate

**PASS — no blockers, no majors.** All seven DAO query sites bind every value
with `$1`/`$2`; no value or identifier is ever interpolated into SQL text.
DML/DDL reviewed: 2 migrations, `dao/favorite_dao.go`, the DAO tests, and the
test-DB bootstrap.

Two minor notes, both recorded rather than fixed:

- `tests/testdb/testdb.go:33` — the test container's password is the Postgres
  image's public default on an ephemeral loopback-only throwaway database. Not
  a secret leak; left as-is deliberately. Optional hygiene only.
- `sql_migrations/main/2_favorite_items.sql` — the 500 cap has no schema
  constraint, only the application-level lock-plus-count. Correct and the only
  tractable design for a per-user row count, and safe because this DAO is the
  sole writer of `favorite_items`.

Specifically cleared as correct: the `FOR UPDATE` per-user lock that makes the
count-then-insert check atomic; the EXISTS-before-COUNT ordering that keeps
re-favoriting idempotent at the cap; and the single scoped
`DELETE ... WHERE user_id = $1 AND item_id = $2`, which makes a cross-user
delete impossible by construction rather than by a check (REQ-008).

### Production-readiness gate

**The gate named by the role prompt, `nexcraft:production-readiness-agent`,
does not exist in the installed plugin** — `D:/code/my-local-code/vendor/nexcraft/agents/`
has no such file. The plugin ships a `production-audit` *skill* instead, and
that skill's lens was applied to this diff. Reported here plainly: the named
agent did not run, something else did.

**Score: 72/100 — launchable with caveats.** The code layer is sound; the
score is held down entirely by packaging that Part 6 owns and has not produced
yet, not by a code defect.

Verified present:

- Health check that proves dependency reachability: `/v1/health` via
  nexcommon's `health.HealthEndpoint`.
- `/metrics` scraped outside user-token validation, and status-aware — the
  hardcoded-200 gap the architecture doc warned about is not present.
- Migrations run forward and each has a `-- +migrate Down`, so there is a
  rollback path.
- Logger initialized as the literal first line of `config`'s `init()`, before
  `envconfig.Process` — the silent-`log.Fatal` trap is avoided.
- No secret or populated-config dump anywhere; the only startup log line names
  `application` and `port`.

Caveats, in order of weight:

1. **No `Dockerfile` and no `docker-compose.yml`.** ~~The deploy surface does
   not exist yet.~~ **Resolved in Part 6** — both now exist and the stack was
   brought up end-to-end; see *Delivery guard (Part 6)*. The Part 5 score was
   taken against the diff at that time and is left as recorded.
2. **No CI workflow.** `.github/workflows/` is absent. Per the audit's own
   rubric a green CI is assumed by a "strong" score; there is none to be green.
3. **`config/config.go:70` references `sql_migrations/seed`, which does not
   exist.** ~~Nothing in the repo provides it.~~ **Resolved in Part 6** — the
   fixture now exists at `sql_migrations/seed/1_sandbox_seed.sql`.
4. **`-race` could not run locally** (see Known limitations). CI should run the
   cap's concurrency test under the race detector.

### Spec compliance

Point-by-point against `spec.md` v3.

| Spec item | Status | Where |
|---|---|---|
| REQ-001 favorite | met | `POST /v1/favorites` → `favorite.Service.Add` |
| REQ-002 list | met | `GET /v1/favorites` → `favorite.Service.List` |
| REQ-003 un-favorite | met | `DELETE /v1/favorites/{item_id}` → `favorite.Service.Remove` |
| REQ-004 discontinued still listed, marked unavailable | met | `List` reads `catalog_items.is_available` live at list-time; no denormalized copy |
| REQ-005 idempotent favorite | met | `INSERT ... ON CONFLICT (user_id, item_id) DO NOTHING`; the `AddAlreadyFavorited` outcome is a success |
| REQ-006 anonymous cannot favorite | met (middleware) | `UserAccessValidator()` runs before every handler; the FE redirect is out of backend scope (see Test case feedback) |
| REQ-007 500 cap, clear error, no silent fail/evict | met | `MaxFavoritesPerUser = 500`; 422 `FAVORITES_LIMIT_REACHED`; the transaction inserts nothing and evicts nothing |
| REQ-008 uniform 404 on mismatch-or-missing | met by construction | one scoped query per read/write; the `dto/in` body carries no user field at all |
| One-per-(user, item) | met | composite primary key, the real guarantee |
| Max 500 per user | met | see REQ-007 |
| Row-ownership on every read/write | met | user id read from `ctx.AuthAccessTokenModel.AuthenticationServerUserID`, never from the request |
| Non-goals: no sharing, no recommendations | met | no such field or endpoint exists |
| Deferred: sort/filter | respected | `GET /v1/favorites` takes no filter/order params; the DAO applies recency ordering only |
| Dependency: catalog items (read-only) | met | `ItemExists` + the `List` join; no write path to `catalog_items` |
| Dependency: auth/session (consumed) | met | user id comes from the token model; no session logic here |
| External integrations: none | met | no network boundary added |

No spec item is unmet. The four recorded deviations are spelling/shape
differences (route prefix, the library's error-envelope field names, plain
`WrapService` for the list, and the non-positive `item_id` → 404 choice), all
argued in Deviations above.

### Architecture compliance

- **Components.** `favorite_endpoint`, the favorite service, and `favorite_dao`
  all exist as the doc names them. The service and DAO packages are
  `services/favorite` and `dao` rather than literals `favorite_service` /
  `favorite_dao` — that is the house package layout, and the doc's names are
  component names, not file paths.
- **Authorization.** Every route is behind a validator; there is no
  role/permission distinction beyond "logged in", exactly as the doc states, so
  no `.Permisson(...)`/`.APIScope(...)` is set. No endpoint addresses a favorite
  by anything but the caller's own `(user_id, item_id)`.
- **Data design.** `favorite_items (user_id, item_id, created_at)`, composite
  PK, no surrogate key, no soft-delete columns — matches `database.md`'s Batch 1
  exactly, including its argued deviations from the house table convention.
- **`spec_traceability`.** Every row of the table is answered by the
  implementation. There is no row whose architectural answer is unimplemented.
- **`## Observability`.** The doc's decision was "wire standard request metrics
  once, and do not accept a status-blind wrapper." The implementation is
  status-aware: `middleware/metrics.go` captures the real response status into
  `my_app_http_hist` and labels by route template, and its behaviour is pinned
  by `tests/middleware` including the negative cases (a 422 is not folded into
  200; an unmatched request adds no series). No feature-specific dashboard or
  counter was invented, as the guard requires. The doc's named verification — a
  manual `/metrics` scrape confirming a 2xx and a 422 land on different labels —
  is covered by `TestScrapeSeparatesSuccessFromCapRejection`, and the live
  scrape is Part 6.

### Delivery guard (Part 6)

**Scope.** Packaging and API artifacts, plus a live end-to-end run of the whole
service — not just the handler layer. Diff from the Part 5 checkpoint:

| Path | What it is |
|---|---|
| `Dockerfile` | Multi-stage, `-mod=vendor`, non-root `app` user, `i18n/` copied into the runtime stage (the bundles are data, not compiled in). |
| `docker-compose.yml` | Postgres 16 + Redis 7 + the app, both dependencies gated by `condition: service_healthy`. |
| `sql_migrations/seed/1_sandbox_seed.sql` | The fixture `config/config.go:70` points at; idempotent. |
| `api/openapi.yaml` | OpenAPI 3.1, hand-authored (see the generator bug below). |
| `postman/sandbox-svc.postman_collection.json` | Generated from the above. |
| `postman/local.postman_environment.json` | `baseUrl` + `apiKey` + `item_id`. |
| `postman/enrich-collection.js` | Deterministic post-processor re-applied after every regeneration. |

**Two tool defects worked around, both recorded rather than hidden.**

1. **The Swagger generator drops a route.** `NewHandleFuncParam` does
   `SwaggerContainer.SwaggerPath[path] = make(...)` on every call, so two routes
   sharing a path overwrite each other and only the last survives. `POST
   /v1/favorites` is silently missing from the generated document. `api/openapi.yaml`
   is therefore hand-authored and validated with `redocly lint` (*"Your API
   description is valid"*, two intentional warnings: no license, localhost
   server).
2. **Postman's bearer helper cannot talk to this service.** `UserAccessValidator`
   uses the *raw* `Authorization` value as the Redis key — it is not a JWT and is
   never parsed. Postman's `http/bearer` auth prepends `"Bearer "`, which would
   look up the wrong key and 401 every request. The security scheme is modelled
   as `apiKey`-in-header instead, so the header is sent verbatim, with the reason
   written into the spec.

**The live run found a real bug — fixed, and the guard that missed it is now the
guard that catches it.**

An unauthenticated request returned:

```json
{"payload":{"status":401,"code":"E-1-CMD-AUT-001","message":"E-1-CMD-AUT-001"}}
```

The client received the raw error code as its message. `i18n/common/error/en-US.json`
and `error/error.go`'s code list carried an **invented** `E-3-SBX-AUT-001`, while
nexcommon actually emits `E-1-CMD-AUT-001` (`vendor/.../error/message.go:5`). The
bundle-coverage guard passed because it only checked the codes in its own — wrong —
list.

Fixed on all three fronts:

- The bundle now carries all 22 codes nexcommon's `error/message.go` can emit, plus
  this service's own three and the formator's configured default.
- `BundledCodes` no longer hand-lists nexcommon's codes at all.
- `tests/bundle_coverage_test.go` gained `TestEveryNexcommonErrorCodeIsBundled`,
  which **reads the codes out of the vendored `message.go`** and asserts each is
  bundled — so the list cannot drift from the library again. Mutation-checked:
  renaming `E-1-CMD-AUT-001` in the bundle makes it fail with
  `missing key "E-1-CMD-AUT-001"`; restoring it passes.

After the rebuild the same request returns `"message":"Unauthorized"`.

**Live end-to-end results.** Built, `docker compose up`, all three containers
healthy, migrations applied by the app on boot, Postgres seeded from the fixture,
two Redis tokens seeded by hand (the fixture covers Postgres only) in the exact
`AuthAccessTokenModel` JSON shape.

| Check | TC | Result |
|---|---|---|
| Empty list is `{"favorites":[]}`, not `null` | TC-004 | 200, `[]` |
| No token → 401 | TC-010 | 401, `E-1-CMD-AUT-001`, message `Unauthorized` |
| Discontinued item still listed, `available:false` | TC-007 / REQ-004 | 200, item 3 present |
| User 2 deleting user 1's row → uniform 404, row intact | TC-016 / REQ-008 | 404 `FAVORITE_NOT_FOUND`; user 1's row unchanged |
| Add at 500 cap → clear 422, nothing inserted, nothing evicted | REQ-007 | 422 `FAVORITES_LIMIT_REACHED`; user 1's count 501 → 501, item 2 absent |
| All three endpoints | TC-001…003 | Newman: 3 requests, 6 assertions, 0 failures |

**The observability decision in `architecture.md` is verified live.** The doc
mandated status-aware metrics and named this exact verification — a scrape
showing a 2xx and a 422 on different labels:

```
my_app_http_hist_count{method="GET", path="/v1/favorites",  status="200"} 1
my_app_http_hist_count{method="GET", path="/v1/favorites",  status="401"} 1
my_app_http_hist_count{method="POST",path="/v1/favorites",  status="422"} 1
my_app_http_hist_count{method="GET", path="/v1/health",     status="200"} 4
```

A 200 and a 422 land on different labels; the label is the route template, not an
ID; `/metrics` does not count itself. No feature-specific dashboard or counter was
invented.

Stack brought down with `docker compose down`. The `18080:8080` host mapping is
deliberate: this machine already has another stack on 8080, and the container port
stays the service's real one.

**Limitations, updated.**

- **`-race` still cannot run locally** (no gcc on this machine), so the 500-cap
  concurrency guarantee remains unit-tested but not race-detector-verified. CI
  should run it under `-race`.
- **No CI workflow** (`.github/workflows/` absent), unchanged from Part 5.
- The Redis token fixture is still seeded by hand; the Postgres fixture now exists.

## New-standard re-check (post-Part-6)

The `nexcraft` plugin standard was updated after this feature landed: three
mechanical scripts now exist under the plugin's `scripts/` (`detect_go_project_layout.py`,
`verify_generated_go_layout.py`, `verify_bundle_coverage.py`) where before there
were none. `/go-dev`'s Part 4 now requires running the verifiers after any point
that wrote Go files, and states they exist *because* of the two failures this
repo hit — an endpoint file at the module root and an invented auth error code.
This section re-audits the committed feature against them.

### `verify_generated_go_layout.py` — FAILED, then fixed

First run reported:

```
FAILED — the following files must be fixed before this task is done:
  - endpoint file not inside a '*_endpoint/' directory:
    server_config\server_endpoint.go (parent dir: 'server_config')
```

`server_config/server_endpoint.go` is the composition root's route-registration
file (`InitEndpoint()`), not an entity endpoint — the check's name-suffix
heuristic can't tell those apart, and it is right that it can't: the scaffold
this project follows names the composition root's endpoint file `endpoint.go`.
Renamed the three composition-root wiring files to that scaffold's canonical
names — `server_dao.go` → `dao.go`, `server_services.go` → `service.go`,
`server_endpoint.go` → `endpoint.go`. Pure renames: same package, same exported
symbols, no import changes.

Re-run: **OK — 28 generated files, no misplaced endpoint files, no leaked
template artifacts.**

### `verify_bundle_coverage.py` — cannot run against the pinned nexcommon

```
ERROR: `go run` failed while collecting registered codes ...
.\tmp..._verify_bundle_coverage_check.go:11:27: undefined: nexerr.AllCodes
```

The script derives the required code set from `error.AllCodes()`, a function
added to nexcommon *for this purpose*. The pinned `v0.3.57-swagger-alpha` predates
it — the vendored `error` package exports only `NewErrorFormator`,
`NewUnBundledErrorMessages` and `NewErrorMessageParam`. The script therefore
exits 2 before it can check anything; wiring it up would need a nexcommon bump,
which is a dependency decision outside this feature's code.

The equivalent coverage is held locally by the guard tests, which together cover
exactly the two halves `AllCodes()` would return:

| Source | Mechanism | Count |
|---|---|---|
| nexcommon's own sentinels | `TestEveryNexcommonErrorCodeIsBundled` derives codes from the vendored `error/message.go` by regex | 22 |
| this service's sentinels + the formator default | `TestEveryErrorCodeIsBundled` walks `error.BundledCodes` | 4 |

All four pass; the bundle carries 26 keys. The derivation is deliberately from
the vendored source rather than a hand-written list, which is the same property
`AllCodes()` gives the script — see the Part 6 entry above for the bug that
proved why.

### `detect_go_project_layout.py` — reconnaissance, two real gaps closed

Advisory only (it names conventions; it does not gate). It reported three items
this pass could act on, and two that are its own gaps:

- **`sql_migrations/`: not present** → fixed. The repo used `migrations/`; the
  house layout names `sql_migrations/` and the feature docs are silent on it, so
  the house name applies. Renamed the directory (keeping the internal
  `main/` + `seed/` split) and updated every reference: `main.go`, the DAO test
  helper, both `Dockerfile` stages, `docker-compose.yml`'s seed mount and the
  `config.go` comment. Detector now reports `sql_migrations/: subfolder-per-target
  shape, targets: main, seed`.
- **Kong statement: not found** → added to `CLAUDE.md`. The service is not behind
  Kong, so every route uses the plain `UserAccessValidator()`; that is now
  written down rather than left for the next reader to infer.
- **Validator usage: "none of the known validator names matched"** → a false
  positive. The detector's name list omits the plain `UserAccessValidator`, which
  the pinned nexcommon *does* provide (`controller/user_access.go:47`) and which
  all three routes use. No code change; the `CLAUDE.md` line above now answers it.
- **`service/`: not present** and **composition root not found** → also its own
  gaps: this repo uses `services/` (the name the layout skill gives) and
  `server_config/` (which the config standard names as a real-project convention).
  Neither is a deviation.

### Guard

`gofmt` clean; `go vet -mod=vendor ./...` and `go build -mod=vendor ./...` pass;
`go test -mod=vendor ./tests/...` all green, including `tests/db` (7.6s against a
real Postgres — which also re-verifies the renamed migration path the test helper
globs).
