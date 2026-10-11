---
feature_code: FEAT-001
service: sandbox-svc
ticket_url: null
version: 3
status: complete
completeness:
  problem: covered
  target_user: covered
  success_criteria: covered
  non_goals: covered
  primary_flow: covered
  edge_cases: covered
  dependencies: covered
  priority: covered
  data_validation_rules: covered
  external_integrations: covered
open_questions: []
sources: []
architect_feedback: []
pr_history:
  - pr_url: https://github.com/william22913/sandbox-docs/pull/1
    version: 1
  - pr_url: https://github.com/william22913/sandbox-docs/pull/3
    version: 2
pr_url: https://github.com/william22913/sandbox-docs/pull/7
last_updated: 2026-10-10

---

# FEAT-001 — Favorite Items (sandbox-svc)

## Problem

Users browsing the catalog have no way to quickly get back to items they're interested in. They have to re-search every time, which is annoying for anyone comparing multiple items before buying.

## Target User

Any logged-in user browsing the catalog.

## Primary Flow

- **REQ-001**: A logged-in user viewing an item detail page can tap a "favorite" star icon to add the item to their personal favorites list.
- **REQ-002**: There is a "My Favorites" page that lists everything the user has favorited.
- **REQ-003**: The user can un-favorite an item from either the item detail page or the favorites list.

## Non-Goals

- No sharing favorites with other users.
- No recommendation engine based on favorites.
- This is just a personal saved list.
- Sorting or filtering the favorites list is explicitly deferred to a later pass (see Priority).

## Edge Cases

- **REQ-004**: If a favorited item is later discontinued/removed from the catalog, it still shows up in the favorites list but marked "unavailable" rather than erroring or silently vanishing. Clarified per catalog-service team confirmation: `catalog_items` rows are **never hard-deleted** — a discontinued/removed item is only ever soft-marked (e.g. a discontinued/unavailable flag), the row itself always persists. This confirms that a live join from `favorite_items` to `catalog_items` (reading current availability at list-time, not denormalizing a copy of item fields at favorite-time) correctly satisfies this requirement — there is no hard-delete case where the row could silently disappear from the join. (source: catalog-service team confirmation, per user, 2026-10-10)
- **REQ-005**: Favoriting an already-favorited item is idempotent — no duplicate entries, no error.
- **REQ-006**: Anonymous (not-logged-in) users cannot favorite at all; attempting to favorite redirects them to login.
- **REQ-007**: A user cannot exceed 500 favorites. Attempting to favorite a 501st item returns a clear error; it must not silently fail or silently evict an older favorite.
- **REQ-008**: The API must verify that the authenticated user's id matches the `favorite_items` row's owning user id before allowing any read or write on that row (viewing it, un-favoriting it, or — per FEAT-002's dependency on this table — editing/clearing its note). On a mismatch, or if the row doesn't exist at all, the API returns a consistent `404 Not Found` in both cases — never `403`, and never any response shape that lets a caller distinguish "this row belongs to someone else" from "this row doesn't exist," so the endpoint can't be used to enumerate other users' favorites. (confirmed by user this session, raised originally by /testcase-generation while designing FEAT-002's test cases — see FEAT-002 spec's Dependencies section)

## Dependencies

- Existing catalog items table/service (to resolve item details and current availability/discontinued status).
- Existing user auth/session system (to identify the logged-in user, gate anonymous access, and supply the authenticated user id that REQ-008's ownership check compares against).

## Priority

This pass covers:
- Favorite / un-favorite (REQ-001, REQ-003, REQ-005, REQ-006)
- The plain "My Favorites" list view (REQ-002, REQ-004)

Explicitly deferred to a later pass:
- Sorting or filtering the favorites list.

## Success Criteria

- A favorited item appears on the "My Favorites" list immediately, without a page reload.
- An un-favorited item disappears from the "My Favorites" list immediately, without a page reload.
- No post-launch business metric is required for this pass — none, because this is an internal UX convenience feature with no dedicated analytics requirement for this pass.

## Data Validation Rules

- One-favorite-per-(user, item): idempotent, no duplicate entries, no error (REQ-005).
- Max 500 favorites per user: attempting to exceed it returns a clear error; it must not silently fail or evict an older favorite (REQ-007).
- Row-ownership check on every `favorite_items` read/write: authenticated user id must match the row's owning user id, else a uniform 404 (REQ-008).

## External Integrations

None, because the catalog items service and the auth/session system are internal in-process packages within this same sandbox-svc codebase (same monolith, no HTTP/gRPC boundary) — there is no real network-exposed external integration here.

## Feedback responses

- CPF-001: addressed in v2 (REQ-004) — confirmed with the catalog-service team that `catalog_items` rows are never hard-deleted, only soft-marked discontinued; the existing live-join architecture is correct as-is, and REQ-004's wording was clarified to state this explicitly so the question doesn't resurface.
- Open question from /testcase-generation (via FEAT-002's `testcase.md` draft, branch `feat/sandbox-svc/FEAT-002/add-favorite-notes`): no REQ-* governed per-user row ownership on `favorite_items`, so FEAT-002's non-goal "notes are never visible to other users" had no enforceable rule behind it. Addressed in v3 (REQ-008) — the ownership check is stated here in FEAT-001, since it governs `favorite_items` generally (viewing, un-favoriting, and FEAT-002's note edit/clear), not just notes. FEAT-002's spec now cites this REQ as a dependency.
