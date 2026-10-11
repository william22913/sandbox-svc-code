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
