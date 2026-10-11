---
feature_code: FEAT-001
service: sandbox-svc
version: 3
status: complete
spec_version: 3
open_questions: []
pr_history:
  - pr_url: https://github.com/william22913/sandbox-docs/pull/2
    version: 1
  - pr_url: https://github.com/william22913/sandbox-docs/pull/4
    version: 2
pr_url: https://github.com/william22913/sandbox-docs/pull/11
last_updated: 2026-10-11
---

# FEAT-001 — Favorite Items — Test Cases

## Requirement coverage

| Requirement | Description | Test cases | Notes |
|---|---|---|---|
| REQ-001 | Logged-in user can tap favorite star on item detail page to add to favorites | TC-001, TC-002 | |
| REQ-002 | "My Favorites" page lists everything the user has favorited | TC-003, TC-004 | |
| REQ-003 | User can un-favorite from item detail page or favorites list | TC-005, TC-006 | |
| REQ-004 | Discontinued/removed favorited item still shows in list, marked "unavailable" | TC-007, TC-008 | Reviewed against spec_version 2's wording clarification (catalog_items rows are never hard-deleted, only soft-marked discontinued) — TC-007/TC-008 already describe exactly this live-join/soft-mark behavior; no case changes required |
| REQ-005 | Favoriting an already-favorited item is idempotent (no duplicates, no error) | TC-009 | |
| REQ-006 | Anonymous users cannot favorite; redirected to login | TC-010, TC-011 | |
| REQ-007 | Cannot exceed 500 favorites; 501st attempt returns clear error, no silent fail/evict | TC-012, TC-013, TC-014 | |
| REQ-008 | Row-ownership enforcement on favorite_items; uniform 404 on mismatch-or-missing | TC-015, TC-016 | New in spec_version 3 — see Refresh log |
| Success criteria (immediate list update, no reload) | Favorite/un-favorite reflected on "My Favorites" list immediately | TC-002, TC-006 | Covered as expected-result assertions within the add/remove cases |
| Non-goal: sharing with other users | Not testable as a positive feature; confirm absence | — | Not testable — spec explicitly excludes this from scope; no behavior to verify beyond its absence, which is out of scope for black-box test design this pass |
| Non-goal: recommendation engine | Not testable as a positive feature | — | Not testable — explicitly out of scope for this pass |
| Non-goal: sort/filter on favorites list | Deferred to later pass | — | Not testable — explicitly deferred; no current behavior to verify |

## Test cases

### TC-001 — Favorite an item from the item detail page (positive)
- **Type**: positive
- **Spec reference**: REQ-001
- **Preconditions / test data**: User is logged in with a valid session. Item X exists in the catalog and is not currently in the user's favorites.
- **Action/input**: On item X's detail page, tap the "favorite" star icon.
- **Expected result**: The star icon changes to its "favorited" state. Item X is now present in the user's favorites (verifiable via the "My Favorites" page, see TC-003/TC-004). No error is shown.
- **Priority**: High

### TC-002 — Favorited item appears on "My Favorites" list immediately without reload (positive)
- **Type**: positive
- **Spec reference**: REQ-001; Success Criteria (immediate list update, no reload)
- **Preconditions / test data**: User is logged in and currently viewing the "My Favorites" page in one context (e.g., open in another tab/view), or will navigate to it immediately after favoriting. Item Y is not yet favorited.
- **Action/input**: Favorite item Y (e.g., from its detail page), then check the "My Favorites" page without a manual page reload/refresh.
- **Expected result**: Item Y appears in the "My Favorites" list immediately, without requiring a full page reload.
- **Priority**: High

### TC-003 — "My Favorites" page lists all favorited items (positive)
- **Type**: positive
- **Spec reference**: REQ-002
- **Preconditions / test data**: User is logged in and has previously favorited 3 distinct catalog items (A, B, C), none discontinued.
- **Action/input**: Navigate to the "My Favorites" page.
- **Expected result**: All 3 items (A, B, C) are listed, each showing recognizable item details (e.g., name/identifier), with none omitted or duplicated.
- **Priority**: High

### TC-004 — "My Favorites" page is empty when user has no favorites (boundary)
- **Type**: boundary
- **Spec reference**: REQ-002
- **Preconditions / test data**: User is logged in and has never favorited any item (or has un-favorited all previously favorited items).
- **Action/input**: Navigate to the "My Favorites" page.
- **Expected result**: The page loads successfully showing zero favorited items (an empty list state), with no error.
- **Priority**: Medium

### TC-005 — Un-favorite an item from the item detail page (positive)
- **Type**: positive
- **Spec reference**: REQ-003
- **Preconditions / test data**: User is logged in; item X is currently in the user's favorites.
- **Action/input**: On item X's detail page, tap the "favorite" star icon again (toggle off).
- **Expected result**: The star icon returns to its "not favorited" state. Item X is removed from the user's favorites list.
- **Priority**: High

### TC-006 — Un-favorite an item from the "My Favorites" list, list updates immediately (positive)
- **Type**: positive
- **Spec reference**: REQ-003; Success Criteria (immediate list update, no reload)
- **Preconditions / test data**: User is logged in and on the "My Favorites" page; item Z is listed there.
- **Action/input**: From the "My Favorites" list, trigger the un-favorite action for item Z.
- **Expected result**: Item Z disappears from the "My Favorites" list immediately, without a page reload.
- **Priority**: High

### TC-007 — Favorited item that becomes discontinued is shown as "unavailable" (edge case)
- **Type**: negative/edge
- **Spec reference**: REQ-004
- **Preconditions / test data**: User has favorited item D. Item D is subsequently discontinued/removed from the catalog (via the catalog items dependency).
- **Action/input**: Navigate to the "My Favorites" page.
- **Expected result**: Item D still appears in the favorites list but is marked "unavailable" (or equivalent indicator). No error is thrown, and item D is not silently dropped from the list.
- **Priority**: High

### TC-008 — Un-favoriting a discontinued ("unavailable") item from the favorites list still works (edge case)
- **Type**: negative/edge
- **Spec reference**: REQ-004; REQ-003
- **Preconditions / test data**: Item D is in the user's favorites and is marked "unavailable" (discontinued).
- **Action/input**: From the "My Favorites" list, trigger the un-favorite action on item D.
- **Expected result**: Item D is removed from the favorites list without error, despite being unavailable/discontinued.
- **Priority**: Medium

### TC-009 — Favoriting an already-favorited item is idempotent (boundary)
- **Type**: boundary
- **Spec reference**: REQ-005
- **Preconditions / test data**: User is logged in; item X is already in the user's favorites.
- **Action/input**: Tap the "favorite" star icon on item X's detail page again (attempt to re-favorite an already-favorited item).
- **Expected result**: No error is returned. No duplicate entry is created for item X. The "My Favorites" list still shows item X exactly once.
- **Priority**: High

### TC-010 — Anonymous (not logged-in) user cannot favorite an item (negative)
- **Type**: negative
- **Spec reference**: REQ-006
- **Preconditions / test data**: No active user session (anonymous/not logged in). Viewing item X's detail page.
- **Action/input**: Tap the "favorite" star icon on item X's detail page while not logged in.
- **Expected result**: The action does not add item X to any favorites list. The user is redirected to the login page/flow instead of the favorite being applied.
- **Priority**: High

### TC-011 — Anonymous user attempting to favorite is redirected to login, then can complete the original intent is out of scope (negative/boundary of spec)
- **Type**: negative
- **Spec reference**: REQ-006
- **Preconditions / test data**: No active session; viewing item X's detail page.
- **Action/input**: Tap the favorite star as an anonymous user, observe the redirect target.
- **Expected result**: Redirect target is the login page/flow (not an error page, not a silent no-op with no feedback). No favorite record is created for any user as a result of this attempt.
- **Priority**: Medium

### TC-012 — User can favorite up to exactly 500 items (boundary)
- **Type**: boundary
- **Spec reference**: REQ-007
- **Preconditions / test data**: User is logged in and currently has exactly 499 favorited items.
- **Action/input**: Favorite one more (not-yet-favorited) item, bringing the total to 500.
- **Expected result**: The 500th favorite is accepted successfully; no error. The "My Favorites" list now shows 500 items.
- **Priority**: High

### TC-013 — Attempting to favorite a 501st item returns a clear error (negative/boundary)
- **Type**: negative
- **Spec reference**: REQ-007
- **Preconditions / test data**: User is logged in and currently has exactly 500 favorited items (at the cap).
- **Action/input**: Attempt to favorite one additional, not-yet-favorited item (the 501st).
- **Expected result**: A clear error is returned to the user indicating the favorites limit has been reached. The attempted item is NOT added. The existing 500 favorites remain unchanged (no silent eviction of any existing favorite, no silent failure that looks like success).
- **Priority**: High

### TC-014 — Un-favoriting an item while at the 500-item cap frees a slot for a new favorite (boundary)
- **Type**: boundary
- **Spec reference**: REQ-007; REQ-003
- **Preconditions / test data**: User is logged in and currently has exactly 500 favorited items (at the cap).
- **Action/input**: Un-favorite one existing item (bringing the count to 499), then favorite a new, previously-unfavorited item.
- **Expected result**: The un-favorite succeeds, dropping the count to 499. The subsequent favorite of the new item succeeds without error, bringing the count back to 500.
- **Priority**: Medium

### TC-015 — Un-favoriting an item that was never favorited by this user returns 404 (negative)
- **Type**: negative
- **Spec reference**: REQ-008
- **Preconditions / test data**: User A is logged in. Item X exists in the catalog but is not in user A's favorites (whether or not it is favorited by any other user is irrelevant and not set up as a precondition — see TC-016 for the specific cross-user case).
- **Action/input**: User A attempts to un-favorite item X (e.g. `DELETE /favorites/{item_id}` for item X).
- **Expected result**: The API returns `404 Not Found`. No favorite record for any user is altered. The response does not distinguish "never favorited by anyone" from any other not-found case.
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

### TC-016 — Un-favoriting another user's favorited item returns the same 404, never a different status (negative, row-ownership)
- **Type**: negative
- **Spec reference**: REQ-008
- **Preconditions / test data**: User B has favorited item Y. User A (a different, logged-in user) has not favorited item Y.
- **Action/input**: User A attempts to un-favorite item Y.
- **Expected result**: The API returns `404 Not Found` — the exact same status and body shape as TC-015's "never favorited by anyone" case, never `403 Forbidden` and never any response distinguishable from TC-015. User B's favorite of item Y is completely unaffected (still present after the attempt).
- **Level**: API
- **Automation candidate**: yes
- **Priority**: High

## Refresh log

Revision from spec_version 2 to spec_version 3 — diffing the two
approved spec.md revisions:

- **REQ-008 (new)**: added in spec v3 — row-ownership enforcement on
  `favorite_items`, with a uniform 404 on both "belongs to someone
  else" and "doesn't exist at all." Added TC-015 (never-favorited case)
  and TC-016 (cross-user case) specifically to assert both paths land
  on the exact same response, since that indistinguishability is the
  actual requirement, not just "returns 404 sometimes." This is a real,
  exercisable risk here (REQ-008 exists precisely because the previous
  design had no stated behavior for it) — not a case invented without
  a requirement behind it.
- **REQ-001 through REQ-007**: unchanged in spec v3 (see spec.md's own
  diff — only REQ-008 and its Dependencies/Priority cross-references
  were added). No existing test case content needed revision; TC-001
  through TC-014 still hold exactly as designed.

## Open questions for /document-preparation

None at this time — the spec's primary flow, edge cases, and data validation rules provide concrete, defensible expected results for all designed cases.
