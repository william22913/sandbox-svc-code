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
| 1 | Schema + migrations (`favorite_items`, `users`, `catalog_items`) | `migrations/main/*.sql` | `tests/db` (they run against it) |
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
- `migrations/main/2_favorite_items.sql` — the 500 cap has no schema
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

1. **No `Dockerfile` and no `docker-compose.yml`.** The deploy surface does not
   exist yet. Part 6's `nexcraft:nexcommon-go-docker-standards` step produces
   them; until then this service cannot be built into an image.
2. **No CI workflow.** `.github/workflows/` is absent. Per the audit's own
   rubric a green CI is assumed by a "strong" score; there is none to be green.
3. **`config/config.go:70` references `migrations/seed`, which does not
   exist.** The comment describes a fixture that pre-populates the sandbox
   Redis token. Nothing in the repo provides it, which is also why Part 6's
   live-stack run needs a token seeded by hand.
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
