---
feature_code: FEAT-005
service: sandbox-svc
version: 1
status: complete
spec_version: 1
open_questions: []
pr_url: https://github.com/william22913/sandbox-docs/pull/24
last_updated: 2026-11-08
---

# FEAT-005 — Favorites Summary and Badge Tooltip — Test Cases

Black-box test design derived from the merged `spec.md` (version 1) and the
FEAT-001/FEAT-004 requirements it depends on. These are designed cases for
later execution, not results.

## Requirement coverage

| Requirement | Description | Test cases | Notes |
|---|---|---|---|
| REQ-001 | Signed-in client calls `GET /v1/favorites/summary`; service returns a summary of the authenticated user's `favorite_items` rows | TC-001 | |
| REQ-002 | Success is `200` + `application/json` + `{"total": <integer>, "discontinued": <integer>}`; zero favorites is a valid `200` with both `0` | TC-001, TC-002, TC-003 | |
| REQ-003 | `total` counts all owned rows, including discontinued catalog items; no availability filter | TC-001, TC-003, TC-004 | TC-004 is the mixed available/discontinued partition |
| REQ-004 | `discontinued` counts owned rows whose catalog item is currently soft-marked discontinued; `discontinued <= total` | TC-001, TC-004, TC-005, TC-008, TC-014, TC-034 | Includes the `0 discontinued` and all-discontinued partitions |
| REQ-005 | Hovering the visible FEAT-004 badge shows exactly `N favorites (M discontinued)`; exact integers, badge `99+` cap never applies to `N` | TC-016, TC-018, TC-019, TC-020, TC-034 | Exact-template boundaries: `M = 0`, `N = 1`, `N > 99`, `M = N` |
| REQ-006 | Tooltip only when the FEAT-004 badge is visible; hidden badge (zero count, badge loading, badge error, unauthenticated) means no tooltip and no summary fetch | TC-016, TC-023, TC-024, TC-025, TC-026, TC-035 | |
| REQ-007 | Summary fetched on first hover of a visible badge, not on app load; an already-fetched summary is refreshed after a successful favorite/un-favorite; a never-fetched summary stays first-hover-driven | TC-016, TC-017, TC-022, TC-029, TC-032 | |
| REQ-008 | While loading or after a failed fetch, hovering shows no tooltip; badge is unaffected; a later successful fetch allows the tooltip on a subsequent hover | TC-027, TC-028, TC-031 | |
| REQ-009 | `total` and `discontinued` are non-negative integers derived at request time, `0 <= discontinued <= total`; nothing stored, cached, or denormalized | TC-003, TC-006, TC-007, TC-014 | Lower bound via TC-003/TC-002; upper bound via the FEAT-001 500 cap in TC-006; request-time derivation via TC-014 |
| REQ-010 | Both values scoped to the authenticated user's own rows; user id comes only from the auth context; no caller-supplied user id accepted | TC-011, TC-012 | |
| REQ-011 | No request input can influence, widen, or override either count | TC-012, TC-013 | |
| REQ-012 | Signed-in user with zero favorites gets `200 {"total": 0, "discontinued": 0}`, not `404`/empty/error | TC-002, TC-023 | |
| REQ-013 | Unauthenticated caller gets the existing `401` error envelope and no summary body | TC-009, TC-010, TC-026 | |
| REQ-014 | When every favorite is discontinued, `total` and `discontinued` are equal | TC-005, TC-008, TC-034 | |
| REQ-015 | When the badge is hidden, hovering the header area where it would be shows no tooltip and triggers no summary fetch | TC-023, TC-024, TC-025, TC-026, TC-035 | |
| REQ-016 | While loading or on fetch failure the tooltip is not shown at all — no spinner, error icon, or toast — and the badge keeps its FEAT-004 count | TC-027, TC-028 | |
| REQ-017 | Unexpected backend failure returns the standard `500` envelope; frontend treats it like any summary fetch error and shows no tooltip | TC-015, TC-028 | |
| REQ-018 | A summary fetched before a successful favorite/un-favorite is never shown as-is afterwards; until refresh completes, and if refresh fails, no tooltip rather than pre-action values | TC-029, TC-030, TC-031, TC-035 | |
| REQ-019 | Depends on FEAT-001 `favorite_items` and soft-marked catalog discontinued status; no schema change | TC-004, TC-005, TC-006, TC-014 | Behavioral dependence on existing FEAT-001 rows is covered; "no schema change" is an implementation constraint not independently observable black-box |
| REQ-020 | Depends on FEAT-001 REQ-008 row ownership; satisfied by taking the user id only from auth context | TC-011, TC-012 | |
| REQ-021 | Depends on FEAT-004 badge and its visibility/display rules; does not change the badge; a freshly refreshed tooltip agrees with the current badge count | TC-020, TC-021, TC-027, TC-028, TC-029 | Badge remains `99+`/exact and unaffected in each |
| REQ-022 | Depends on the shared auth middleware; `401` for anonymous; no new login logic | TC-009, TC-010 | |
| REQ-023 | Depends on FEAT-004's refresh-after-favorite/un-favorite trigger; the same successful action refreshes an already-fetched summary | TC-029, TC-030 | |
| REQ-024 | Success criterion: `200 {"total": T, "discontinued": D}` for `0 <= D <= T`; `T = 0` returns `{0, 0}` | TC-001, TC-002, TC-004 | |
| REQ-025 | Success criterion: unauthenticated caller gets `401` envelope, no summary body | TC-009 | |
| REQ-026 | Success criterion: no request input can make one user's counts include another user's favorites | TC-011, TC-012, TC-013 | |
| REQ-027 | Success criterion: visible badge hover shows `N favorites (M discontinued)` with exact integers even under a `99+` badge; hidden badge shows no tooltip | TC-016, TC-020, TC-023 | |
| REQ-028 | Success criterion: loading, failed fetch, or `500` means no tooltip and an unchanged badge count | TC-027, TC-028 | |
| REQ-029 | Success criterion: after a successful favorite/un-favorite, the next tooltip reflects updated values, never pre-action ones | TC-029, TC-030 | |
| Non-goal: click-through from badge or tooltip | No navigation/click action | TC-033 | Exercisable only as "clicking does nothing"; no link destination exists |
| Non-goal: polling, realtime push, WebSocket, cross-tab sync | Deferred | — | Not tested — no behavior exists in this pass; only first hover, refresh-after-action, and reload produce a fetch |
| Non-goal: server-side or persisted client-side cache | No cache; summary held in component state until refreshed/reload | TC-014, TC-022 | Request-time derivation and no-refetch-on-repeat-hover are the observable consequences |
| Non-goal: per-item breakdown | Tooltip shows only the two aggregates | TC-016, TC-018, TC-019, TC-020, TC-034 | Exact whole-string assertions leave no room for item titles/ids |
| Non-goal: admin or cross-user summary | No cross-user access | TC-011, TC-012, TC-013 | |
| Non-goal: change to the FEAT-004 badge | No new badge, count, `99+` cap, or badge loading/error change | TC-020, TC-021, TC-027, TC-028 | Badge display asserted unchanged in each |
| Non-goal: new page, settings toggle, notification/inbox, count history | Excluded | — | Not tested — no such surface exists |
| Non-goal: tooltip trigger other than hover | Hover is the only trigger specified | TC-033 | Click does not open or navigate anything |
| Non-goal: localization/pluralization variants | Literal template always, including `(0 discontinued)` and "favorites" at `N = 1` | TC-018, TC-019, TC-034 | |
| Non-goal: tooltip animation/transition | No animation required | — | Not tested — no animation behavior is required in this pass |
| Non-goal: analytics or post-launch metric | None required | — | Not testable — spec states no metric is required |
| Non-goal: new table, column, index, or cache | No schema change | TC-004, TC-006, TC-014 | Implementation constraint; behavior over existing FEAT-001 rows covered |
| Not applicable: hard-deleted catalog item | FEAT-001 REQ-004 states catalog rows are never hard-deleted | — | No hard-delete state exists to exercise; only soft-marked discontinued is testable |

## Test cases

### TC-001 — Signed-in user's summary returns total and discontinued (positive)
- **Type**: positive
- **Spec reference**: REQ-001, REQ-002, REQ-003, REQ-004, REQ-024
- **Preconditions / test data**: User A is signed in with a valid session and owns exactly 3 `favorite_items` rows. Two referenced catalog items are available; one is soft-marked discontinued/unavailable. No other user's rows share A's `user_id`.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials; no query parameters and no body.
- **Expected result**: HTTP `200`; `Content-Type: application/json`; body is exactly `{"total": 3, "discontinued": 1}`; both values are JSON integers (not strings or floats); no additional fields.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-002 — Zero favorites returns `200 {"total": 0, "discontinued": 0}` (boundary)
- **Type**: boundary
- **Spec reference**: REQ-002, REQ-012, REQ-024
- **Preconditions / test data**: User A is signed in and owns zero `favorite_items` rows.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200` (not `404`, not `204`, not an empty body, not an error); `Content-Type: application/json`; body is exactly `{"total": 0, "discontinued": 0}` with JSON integer zeros.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-003 — Exactly one available favorite returns total `1`, discontinued `0` (boundary)
- **Type**: boundary
- **Spec reference**: REQ-002, REQ-003, REQ-009
- **Preconditions / test data**: User A is signed in and owns exactly 1 `favorite_items` row pointing at a currently available catalog item.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 1, "discontinued": 0}`; values are non-negative JSON integers with `discontinued <= total`.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-004 — Mixed available and discontinued favorites partition correctly (boundary)
- **Type**: boundary
- **Spec reference**: REQ-003, REQ-004, REQ-019, REQ-024
- **Preconditions / test data**: User A is signed in and owns exactly 5 `favorite_items` rows. Exactly 2 of the referenced catalog items are soft-marked discontinued/unavailable; the other 3 are available.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 5, "discontinued": 2}` — `total` is not filtered by availability, and `discontinued` counts only the marked rows and remains a subset of `total`.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-005 — All favorites discontinued means total equals discontinued (boundary)
- **Type**: boundary
- **Spec reference**: REQ-004, REQ-014, REQ-019, REQ-024
- **Preconditions / test data**: User A is signed in and owns exactly 3 `favorite_items` rows, and all 3 referenced catalog items are soft-marked discontinued/unavailable.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 3, "discontinued": 3}` — the two counts are equal, and no availability filter removes rows from `total`.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-006 — FEAT-001 maximum of 500 favorites is summarized exactly (boundary)
- **Type**: boundary
- **Spec reference**: REQ-009, REQ-019
- **Preconditions / test data**: User A is signed in and owns exactly 500 `favorite_items` rows — the maximum permitted by FEAT-001 REQ-007 — of which exactly 125 reference soft-marked discontinued catalog items.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 500, "discontinued": 125}`; no off-by-one, truncation, or overflow.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-007 — Counts above 99 are returned exactly, not capped (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005, REQ-009, REQ-027
- **Preconditions / test data**: User A is signed in and owns exactly 100 `favorite_items` rows, of which 12 reference soft-marked discontinued catalog items.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 100, "discontinued": 12}`. The FEAT-004 `99+` display cap applies only to the badge, never to the summary values (the tooltip consequence is verified in TC-020).
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-008 — A single discontinued favorite returns `{"total": 1, "discontinued": 1}` (boundary)
- **Type**: boundary
- **Spec reference**: REQ-004, REQ-014
- **Preconditions / test data**: User A is signed in and owns exactly 1 `favorite_items` row; its catalog item is soft-marked discontinued/unavailable.
- **Action/input**: Send `GET /v1/favorites/summary` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly `{"total": 1, "discontinued": 1}` — the minimum non-zero case where both counts are equal.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-009 — Unauthenticated request returns `401` and no summary body (negative)
- **Type**: negative
- **Spec reference**: REQ-013, REQ-022, REQ-025
- **Preconditions / test data**: No user session exists; the request carries no credentials.
- **Action/input**: Send `GET /v1/favorites/summary` with no session cookie/token.
- **Expected result**: HTTP `401`; response is the existing error envelope `{"error_code": "<non-empty>", "message": "<non-empty>"}`; the body contains no `total` or `discontinued` fields and no other summary payload.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-010 — Invalid or expired session is treated as unauthenticated (negative)
- **Type**: negative
- **Spec reference**: REQ-013, REQ-022
- **Preconditions / test data**: A malformed or expired session token is supplied.
- **Action/input**: Send `GET /v1/favorites/summary` with the invalid/expired credentials.
- **Expected result**: HTTP `401`; the existing error envelope `{"error_code": "<non-empty>", "message": "<non-empty>"}`; no `total`/`discontinued` payload and no indication of any user's counts.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-011 — Summary is isolated to the authenticated user's own rows (positive)
- **Type**: positive
- **Spec reference**: REQ-010, REQ-020, REQ-026
- **Preconditions / test data**: User A owns 2 `favorite_items` rows (0 discontinued); user B owns 6 rows (4 discontinued). The two users' rows are disjoint.
- **Action/input**: Send `GET /v1/favorites/summary` once with A's session credentials, then once with B's.
- **Expected result**: A's response is exactly `{"total": 2, "discontinued": 0}`; B's is exactly `{"total": 6, "discontinued": 4}`. Neither response includes any of the other user's rows.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-012 — Caller-supplied user identifiers are ignored (negative)
- **Type**: negative
- **Spec reference**: REQ-010, REQ-011, REQ-020, REQ-026
- **Preconditions / test data**: User A is signed in and owns 2 rows (0 discontinued); user B owns 6 rows (4 discontinued).
- **Action/input**: Send `GET /v1/favorites/summary` with A's session, appending `?user_id=<B's id>&total=999&discontinued=999`, plus a caller-supplied `X-User-Id: <B's id>` header.
- **Expected result**: HTTP `200`; body is exactly A's own `{"total": 2, "discontinued": 0}`. The query values, counts, and header do not influence the response and never expose B's data.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-013 — Extra query parameters cannot override either count (negative)
- **Type**: negative
- **Spec reference**: REQ-011, REQ-026
- **Preconditions / test data**: User A is signed in and owns 4 rows (1 discontinued).
- **Action/input**: Send `GET /v1/favorites/summary?total=0&discontinued=0&limit=1000&offset=0` with A's session credentials.
- **Expected result**: HTTP `200`; body is exactly the true `{"total": 4, "discontinued": 1}`. No parameter influences, widens, or overrides the derived counts.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-014 — Counts are derived at request time, not cached (positive)
- **Type**: positive
- **Spec reference**: REQ-004, REQ-009, REQ-019
- **Preconditions / test data**: User A is signed in and owns 3 rows, all referencing currently available catalog items.
- **Action/input**: (1) Send `GET /v1/favorites/summary` with A's session. (2) Soft-mark one of those three catalog items discontinued without changing any `favorite_items` row. (3) Send the same request again.
- **Expected result**: First response is exactly `{"total": 3, "discontinued": 0}`; second response is exactly `{"total": 3, "discontinued": 1}` — the changed availability is reflected on the next request, proving request-time derivation with no stored or denormalized count.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-015 — Unexpected backend failure returns the standard `500` envelope (negative)
- **Type**: negative
- **Spec reference**: REQ-017
- **Preconditions / test data**: A signed-in user with favorites exists; an unexpected internal failure is forced while the summary is being produced (for example, the catalog availability lookup raises an internal error).
- **Action/input**: Send `GET /v1/favorites/summary` with the user's session credentials while the failure is active.
- **Expected result**: HTTP `500`; body is the standard error envelope `{"error_code": "<non-empty>", "message": "<non-empty>"}`; no partial `total`/`discontinued` values are returned.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: Medium

### TC-016 — First hover fetches the summary and shows the exact tooltip text (positive)
- **Type**: positive
- **Spec reference**: REQ-005, REQ-006, REQ-007, REQ-027
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 of them referencing a discontinued catalog item. The FEAT-004 badge is visible and displays `3`. The page has loaded; the summary has never been fetched.
- **Action/input**: Hover the visible FEAT-004 badge.
- **Expected result**: Exactly one `GET /v1/favorites/summary` request is issued on the hover; the tooltip appears with the exact text `3 favorites (1 discontinued)`, matching the response's `total`/`discontinued` integers. The badge remains `3`.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-017 — No summary fetch or tooltip on app load (negative)
- **Type**: negative
- **Spec reference**: REQ-007
- **Preconditions / test data**: User A is signed in with 3 favorites; the FEAT-004 badge is visible.
- **Action/input**: Load the signed-in application and wait without hovering the badge.
- **Expected result**: No `GET /v1/favorites/summary` request occurs during or after app load; no tooltip is displayed. (FEAT-004's own count request may occur and is out of scope here.)
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-018 — Tooltip with zero discontinued renders `(0 discontinued)` (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005, REQ-027
- **Preconditions / test data**: User A is signed in with 3 favorites, all referencing available catalog items; the badge is visible showing `3`.
- **Action/input**: Hover the badge.
- **Expected result**: The tooltip text is exactly `3 favorites (0 discontinued)` — the literal template is used, with no pluralization change and no omission of the zero.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-019 — Tooltip at one favorite uses the literal plural template (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005
- **Preconditions / test data**: User A is signed in with exactly 1 favorite, which is available; the badge is visible showing `1`.
- **Action/input**: Hover the badge.
- **Expected result**: The tooltip text is exactly `1 favorites (0 discontinued)` — no singular "favorite" variant and no localization substitution.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-020 — Tooltip shows the exact total even when the badge shows `99+` (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005, REQ-021, REQ-027
- **Preconditions / test data**: User A is signed in with 150 favorites, 3 of them discontinued; the FEAT-004 badge is visible showing `99+`.
- **Action/input**: Hover the badge.
- **Expected result**: The tooltip text is exactly `150 favorites (3 discontinued)`; the badge continues to display `99+` unchanged.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-021 — Freshly fetched tooltip agrees with the current badge count (positive)
- **Type**: positive
- **Spec reference**: REQ-021
- **Preconditions / test data**: User A is signed in with 42 favorites, 5 of them discontinued; the FEAT-004 badge is visible showing `42`.
- **Action/input**: Hover the badge for the first time.
- **Expected result**: The tooltip text is exactly `42 favorites (5 discontinued)`; the tooltip's `N` equals the badge's displayed count, and the badge is unchanged.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-022 — A repeat hover does not refetch and still shows the held summary (positive)
- **Type**: positive
- **Spec reference**: REQ-007
- **Preconditions / test data**: User A is signed in with 3 favorites (1 discontinued); the badge is visible. The user has already hovered once, and the summary is held in component state.
- **Action/input**: Move the pointer away from the badge, then hover the badge again. Make no favorite/un-favorite action.
- **Expected result**: The tooltip appears again with `3 favorites (1 discontinued)`; no additional `GET /v1/favorites/summary` request is issued, because the summary is fetched on first hover and refreshed only after a favorite/un-favorite or a reload.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-023 — Zero favorites hides the badge, so hovering shows no tooltip and fetches nothing (negative)
- **Type**: negative
- **Spec reference**: REQ-006, REQ-012, REQ-015, REQ-027
- **Preconditions / test data**: User A is signed in and owns 0 favorites; per FEAT-004 the badge is hidden.
- **Action/input**: Hover the header area where the badge would appear.
- **Expected result**: No tooltip is displayed; no `GET /v1/favorites/summary` request is made.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-024 — Badge hidden while the FEAT-004 count is loading: no tooltip, no fetch (negative)
- **Type**: negative
- **Spec reference**: REQ-006, REQ-015
- **Preconditions / test data**: User A is signed in with favorites; the FEAT-004 count request is delayed so the badge is in its hidden loading state.
- **Action/input**: Hover the header area while the count is still loading.
- **Expected result**: No tooltip appears and no `GET /v1/favorites/summary` request is made during the hidden/loading window.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-025 — Badge hidden after a FEAT-004 count error: no tooltip, no fetch (negative)
- **Type**: negative
- **Spec reference**: REQ-006, REQ-015
- **Preconditions / test data**: User A is signed in with favorites; the FEAT-004 count request has failed, so the badge is hidden per FEAT-004 REQ-012.
- **Action/input**: Hover the header area where the badge would appear.
- **Expected result**: No tooltip is displayed; no `GET /v1/favorites/summary` request is made.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-026 — Unauthenticated visitor has no badge, tooltip, or summary request (negative)
- **Type**: negative
- **Spec reference**: REQ-006, REQ-013, REQ-015
- **Preconditions / test data**: The visitor is not signed in.
- **Action/input**: Open the application and hover the header area where the badge would appear.
- **Expected result**: No FEAT-004 badge and no tooltip are shown; no `GET /v1/favorites/summary` request is issued from the page.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-027 — Loading state shows no tooltip or feedback and leaves the badge unchanged (negative)
- **Type**: negative
- **Spec reference**: REQ-008, REQ-016, REQ-021, REQ-028
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 of them discontinued; the badge shows `3`. The summary response is delayed (for example, held for several seconds).
- **Action/input**: Hover the badge while the summary request is in flight and keep it hovered until the response arrives.
- **Expected result**: During loading, no tooltip is shown at all — no spinner, no error icon, no toast; the badge continues to display `3` unchanged. After the response arrives, hovering again shows exactly `3 favorites (1 discontinued)`.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-028 — Failed summary fetch shows no tooltip or error UI; a later success works (negative)
- **Type**: negative
- **Spec reference**: REQ-008, REQ-016, REQ-017, REQ-021, REQ-028
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 of them discontinued; the badge shows `3`. The summary request fails (network error or `500` response).
- **Action/input**: Hover the badge while the summary request fails.
- **Expected result**: No tooltip is displayed and no spinner, error icon, or toast appears; the badge continues to display `3` unchanged. After the summary service is restored and a later fetch succeeds (for example, after a page reload followed by a hover), a subsequent hover shows exactly `3 favorites (1 discontinued)`, with the badge unchanged.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-029 — A successful favorite refreshes the fetched summary without stale values (positive)
- **Type**: positive
- **Spec reference**: REQ-007, REQ-018, REQ-023, REQ-029
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 discontinued; the user has hovered and seen `3 favorites (1 discontinued)`, and the badge shows `3`. A fourth, available item exists that A has not favorited.
- **Action/input**: Successfully favorite the fourth item (from item detail or My Favorites), then hover the badge.
- **Expected result**: The successful action refreshes the badge and the already-fetched summary. While the refresh is in flight, hovering shows no tooltip; the pre-action `3 favorites (1 discontinued)` is never shown again. After the refresh completes, hovering shows exactly `4 favorites (1 discontinued)` and the badge shows `4`.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-030 — A successful un-favorite refreshes the fetched summary without stale values (positive)
- **Type**: positive
- **Spec reference**: REQ-007, REQ-018, REQ-023, REQ-029
- **Preconditions / test data**: User A is signed in with 4 favorites, 1 discontinued; the user has hovered and seen `4 favorites (1 discontinued)`, and the badge shows `4`. One of the available favorites is selected for removal.
- **Action/input**: Successfully un-favorite that available item, then hover the badge.
- **Expected result**: While the refresh is in flight, no tooltip is shown; the pre-action `4 favorites (1 discontinued)` is never shown again. After the refresh completes, hovering shows exactly `3 favorites (1 discontinued)` and the badge shows `3`.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-031 — A failed post-action summary refresh never shows stale values (negative)
- **Type**: negative
- **Spec reference**: REQ-008, REQ-018, REQ-021
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 discontinued; the summary was fetched (`3 favorites (1 discontinued)`) and the badge shows `3`. A fourth available item exists. The summary refresh triggered by favoriting is forced to fail.
- **Action/input**: Successfully favorite the fourth item, then hover the badge while and after the summary refresh fails.
- **Expected result**: The tooltip is not shown at all after the action — the pre-action values are never displayed. The badge follows its own FEAT-004 behavior and is unaffected by the summary failure. When a later summary fetch succeeds (for example, after a page reload followed by a hover), the tooltip shows the current true values (`4 favorites (1 discontinued)`), never the stale pre-action values.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: High

### TC-032 — A never-fetched summary stays first-hover-driven across a favorite action (negative)
- **Type**: negative
- **Spec reference**: REQ-007
- **Preconditions / test data**: User A is signed in with 3 favorites, 1 discontinued; the badge shows `3`; the summary has never been fetched (the badge has not been hovered).
- **Action/input**: Successfully favorite a fourth, available item, then hover the badge for the first time.
- **Expected result**: The favorite action refreshes only the FEAT-004 badge and does not fetch the never-fetched summary. The first hover fetches it and shows exactly `4 favorites (1 discontinued)`; the badge shows `4`.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-033 — Tooltip is hover-only and display-only; clicking does nothing (negative)
- **Type**: negative
- **Spec reference**: REQ-005, non-goal: no click-through
- **Preconditions / test data**: User A is signed in with 3 favorites (1 discontinued); the badge is visible and the summary is available.
- **Action/input**: Hover the badge so the tooltip appears, then click the badge and click the tooltip text. Observe the URL, page content, and any navigation.
- **Expected result**: The tooltip appears only via hover. Clicking the badge or the tooltip performs no navigation, does not open "My Favorites", does not change the route, and triggers no favorite/un-favorite action; the badge and tooltip remain informational.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-034 — All-discontinued favorites produce an equal-count tooltip (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005, REQ-014
- **Preconditions / test data**: User A is signed in with 3 favorites, and all 3 reference soft-marked discontinued catalog items; the badge is visible showing `3`.
- **Action/input**: Hover the badge.
- **Expected result**: The tooltip text is exactly `3 favorites (3 discontinued)` — `N` and `M` are equal and both are exact integers.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

### TC-035 — Un-favoriting the last item hides the badge and suppresses the tooltip (boundary)
- **Type**: boundary
- **Spec reference**: REQ-006, REQ-015, REQ-018
- **Preconditions / test data**: User A is signed in with exactly 1 favorite; the badge shows `1`; the summary has been fetched and shows `1 favorites (0 discontinued)`.
- **Action/input**: Successfully un-favorite that only item, then hover the header area where the badge was.
- **Expected result**: The FEAT-004 badge disappears (count `0`), no tooltip is shown, and no further summary fetch is triggered by the hover — the fetched summary is not displayed for a hidden badge.
- **Level**: UI
- **Automation candidate**: yes
- **Priority**: Medium

## Open questions for /document-preparation

None. Every expected result in this pass is derivable from the merged spec
(version 1) and the FEAT-001/FEAT-004 requirements it cites.
