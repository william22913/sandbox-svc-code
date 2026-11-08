---
feature_code: FEAT-005
service: sandbox-svc
ticket_url: null
has_ui: true
version: 1
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
pr_history: []
pr_url: https://github.com/william22913/sandbox-docs/pull/23
last_updated: 2026-11-08
---

# FEAT-005 — Favorites summary and badge tooltip (sandbox-svc)

## Problem

FEAT-004's header badge shows only one number: the user's total favorite
count. A signed-in user therefore cannot tell, without opening "My
Favorites" and scanning the list, how many of their saved items are
discontinued/unavailable. This feature adds a per-user summary — total
favorites and how many of them are discontinued — and surfaces it as a
tooltip on the existing FEAT-004 badge. (source: user, this session)

## Target User

Any signed-in user of `sandbox-svc` who has the FEAT-004 badge visible
(at least one favorite). Users with zero favorites see no badge, so no
tooltip either. Anonymous visitors see neither the badge nor the
tooltip. (source: user, this session)

## Primary Flow

- **REQ-001**: A signed-in user's client calls
  `GET /v1/favorites/summary`; the service returns a summary of the
  `favorite_items` rows owned by the authenticated user.
- **REQ-002**: A successful response is `200` with
  `Content-Type: application/json` and body
  `{"total": <integer>, "discontinued": <integer>}`. When the user has
  no favorites both values are `0` — a valid `200`, not an error.
- **REQ-003**: `total` is the number of all `favorite_items` rows owned
  by the user, including rows whose catalog item is discontinued —
  consistent with FEAT-001 REQ-004 and FEAT-004 REQ-004.
- **REQ-004**: `discontinued` is the number of the user's
  `favorite_items` rows whose referenced catalog item is currently
  soft-marked discontinued/unavailable per FEAT-001 REQ-004. No other
  availability filter applies, and `discontinued` is always a subset of
  `total`.
- **REQ-005**: Hovering the FEAT-004 header badge, while that badge is
  visible, shows a tooltip whose exact text is
  `N favorites (M discontinued)`, where `N` is the summary's `total` and
  `M` is its `discontinued`, both rendered as exact integers. The
  badge's `99+` display cap (FEAT-004 REQ-011) applies to the badge
  only; it never applies to `N` in the tooltip.
- **REQ-006**: The tooltip is shown only when the FEAT-004 badge itself
  is visible. When FEAT-004 hides the badge (count `0`, badge loading,
  badge fetch error, or unauthenticated — FEAT-004 REQ-010/REQ-012),
  hovering produces no tooltip and triggers no summary fetch.
- **REQ-007**: The summary is fetched on the first hover of a visible
  badge — not on app load. After a successful favorite or un-favorite,
  a summary that has already been fetched is refreshed through the same
  action that refreshes the FEAT-004 badge (FEAT-004 REQ-005); a
  summary that has never been fetched stays first-hover-driven.
- **REQ-008**: While the summary is loading, or after a failed summary
  fetch, hovering shows no tooltip. The badge itself is unaffected — it
  keeps displaying its FEAT-004 count (or staying hidden) exactly as it
  would without this feature. A later successful fetch allows the
  tooltip on a subsequent hover.

(source for REQ-001–REQ-008: user, this session; REQ-005's exact-integer
clause and REQ-007's never-fetched-summary clause are the recommended
resolution of an ambiguity in that brief)

## Data Validation Rules

These rules constrain the response, not a request payload:
`GET /v1/favorites/summary` accepts no request fields, query parameters,
or body.

- **REQ-009**: `total` and `discontinued` are non-negative integers
  derived at request time from the authenticated user's
  `favorite_items` rows, with `0 <= discontinued <= total`. Neither is
  stored, cached, or denormalized.
- **REQ-010**: Both values are always scoped to the authenticated
  user's own rows. The user id comes only from the auth context; no
  caller-supplied user identifier is accepted, so no request input can
  cause another user's favorites to be counted.
- **REQ-011**: No request input can influence, widen, or override either
  count. This is the summary endpoint's application of FEAT-001's
  REQ-008 row-ownership rule (the per-row `404` case does not apply,
  because a summary never addresses an individual row).

(source for REQ-009–REQ-011: user, this session)

## Edge Cases

- **REQ-012**: A signed-in user with zero favorites receives
  `200 {"total": 0, "discontinued": 0}` — not `404`, not an empty body,
  not an error.
- **REQ-013**: An unauthenticated caller receives the existing `401`
  error envelope (`{"error_code": "...", "message": "..."}`) and no
  summary body.
- **REQ-014**: When every favorite is discontinued, `total` and
  `discontinued` are equal (e.g. `{"total": 3, "discontinued": 3}`).
- **REQ-015**: When the FEAT-004 badge is hidden, hovering the header
  area where the badge would be shows no tooltip and does not trigger a
  summary fetch.
- **REQ-016**: While the summary is loading, or if its fetch fails, the
  tooltip is not shown at all — no spinner, no error icon, no toast —
  and the badge continues to display its FEAT-004 count unchanged.
- **REQ-017**: An unexpected backend failure returns the standard `500`
  error envelope; the frontend treats it like any other summary fetch
  error and shows no tooltip.
- **REQ-018**: A summary fetched before a successful favorite or
  un-favorite must never be shown again as-is after that action. Until
  the refresh completes (and if the refresh fails), the tooltip is not
  shown rather than displaying the pre-action `N`/`M` values.

(source for REQ-012–REQ-018: user, this session)

## Dependencies

- **REQ-019**: Depends on FEAT-001's `favorite_items` table and the
  catalog item's soft-marked discontinued/unavailable status
  (FEAT-001 REQ-004); `total` is a count of the user's rows and
  `discontinued` is a count of those rows joined to currently
  discontinued catalog items. No schema change is required by this
  feature.
- **REQ-020**: Depends on FEAT-001's REQ-008 row-ownership rule;
  FEAT-005 satisfies it by taking the user id only from the auth
  context (see REQ-010).
- **REQ-021**: Depends on FEAT-004's header badge and its visibility
  rules (FEAT-004 REQ-003/REQ-010/REQ-011/REQ-012). This feature adds
  hover behavior to that existing badge only; it does not change the
  badge's count source, its exact-count/`99+` display format, its
  visibility rule, or its refresh-after-favorite behavior. The
  summary's `total` and the badge's count both count the same
  `favorite_items` rows for the same user, so a freshly refreshed
  tooltip agrees with the current badge count.
- **REQ-022**: Depends on the shared auth middleware assumed by
  FEAT-001, FEAT-003, and FEAT-004: it places the authenticated user id
  in request context and returns `401` for anonymous callers. FEAT-005
  adds no login/session logic of its own.
- **REQ-023**: Depends on FEAT-004's refresh-after-favorite/un-favorite
  trigger (FEAT-004 REQ-005): the same successful action refreshes an
  already-fetched summary (see REQ-007).

(source for REQ-019–REQ-023: user, this session)

## Non-Goals

- No click-through from the badge or the tooltip to "My Favorites".
- No polling, realtime push, WebSocket, or cross-tab synchronization;
  the summary refreshes on first hover and after a successful
  favorite/un-favorite only.
- No caching: no server-side count cache and no persisted client-side
  cache. The fetched summary is held only in component state until it
  is refreshed or the page is reloaded.
- No per-item breakdown: the tooltip never lists item titles, item ids,
  or which favorites are discontinued; only the two aggregate numbers.
- No admin or cross-user summary.
- No change to FEAT-004's badge: no new badge, no change to its count,
  its `99+` cap, or its own loading/error behavior.
- No new page, settings toggle, notification/inbox behavior, or count
  history.
- No tooltip trigger other than hovering the badge is specified in this
  pass.
- No localization or pluralization variants: the tooltip always uses
  the literal template `N favorites (M discontinued)`, including
  `(0 discontinued)`, and does not switch to a singular "favorite".
- No tooltip animation or transition requirement.
- No analytics or other post-launch business metric.
- No new table, column, index, or cache for the summary.

(source: user, this session)

## Priority

This pass covers the whole feature: the
`GET /v1/favorites/summary` endpoint (REQ-001–REQ-004,
REQ-009–REQ-014), the hover tooltip on the existing FEAT-004 badge with
its visibility, loading, and error behavior (REQ-005, REQ-006, REQ-008,
REQ-015–REQ-017), and first-hover fetch plus refresh-after-
favorite/un-favorite without stale values (REQ-007, REQ-018).

Explicitly deferred to a later pass:
- Click-through from the badge or tooltip to "My Favorites".
- Polling, realtime updates, and cross-tab synchronization.
- Caching (server-side or persisted client-side).
- Per-item breakdown or a discontinued-items list.
- Analytics.
- Localization or pluralization variants.
- Tooltip animation.

(source: user, this session — nothing deferred is needed for this pass)

## Success Criteria

- **REQ-024**: For a signed-in user with `T` total favorites of which
  `D` are discontinued (`0 <= D <= T`), `GET /v1/favorites/summary`
  returns `200` and `{"total": T, "discontinued": D}`; when `T = 0` the
  response is `200 {"total": 0, "discontinued": 0}`. (verifies REQ-001,
  REQ-002, REQ-003, REQ-004, REQ-009, REQ-012, REQ-014)
- **REQ-025**: An unauthenticated caller receives `401` with the
  existing error envelope and no summary body. (verifies REQ-013)
- **REQ-026**: No request input can cause one user's `total` or
  `discontinued` to include another user's favorites. (verifies REQ-010,
  REQ-011)
- **REQ-027**: Hovering the visible FEAT-004 badge shows the tooltip
  `N favorites (M discontinued)` with `N` = `total` and `M` =
  `discontinued`, using exact integers even when the badge displays
  `99+`; hovering when the badge is hidden shows no tooltip. (verifies
  REQ-005, REQ-006, REQ-015)
- **REQ-028**: While the summary is loading, after a failed fetch, or
  after a `500`, hovering shows no tooltip and the badge's own count
  display is unchanged. (verifies REQ-008, REQ-016, REQ-017)
- **REQ-029**: After a successful favorite or un-favorite, the next
  tooltip shown reflects the updated `total` and `discontinued` values,
  never the pre-action ones. (verifies REQ-007, REQ-018, REQ-023)
- No post-launch business metric is required — none, because this is a
  small UX indicator on an existing badge with no dedicated analytics
  requirement for this pass. (source: user, this session)

## External Integrations

None, because the favorites data, the catalog item availability status,
and the auth/session system are internal in-process packages within the
same `sandbox-svc` monolith — same as FEAT-001, FEAT-003, and FEAT-004 —
so there is no network-exposed external integration to specify a
contract for. (source: user, this session)
