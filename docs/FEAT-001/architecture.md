---
feature_code: FEAT-001
service: sandbox-svc
version: 2
status: complete
spec_version: 3
code_repo: https://github.com/william22913/sandbox-svc-code
base_branch: main
dev_feedback: []
code_prep_feedback: []   # CPF-001 (edge_cases/REQ-004) resolved 2026-10-10: spec.md's "Feedback responses" section records "CPF-001: addressed in v2 (REQ-004)" -- v2 is higher than CPF-001's recorded spec_version (1) and no higher than spec.md's current merged version (2), so per Part 4 step 5 this is accepted and removed; confirmed with the catalog-service team that catalog_items rows are never hard-deleted (only soft-marked discontinued), so the existing live-join design (favorite_items -> catalog_items, read at list-time) is correct as-is -- no schema change needed. Full original note in git history (commit cffbb97) if this needs to be found again.
sources: []
pr_url: https://github.com/william22913/sandbox-docs/pull/10
last_updated: 2026-10-11
---

# FEAT-001 — Favorite Items (sandbox-svc) — Architecture

## Revision notes (spec v3)

spec.md v3 added **REQ-008** (row-ownership enforcement on
`favorite_items`), raised by `/testcase-generation` while designing
FEAT-002's test cases and resolved onto FEAT-001 since it governs
`favorite_items` generally, not just FEAT-002's notes field. No stack,
component, or schema change results — see the updated Components
section and REQ-008's row in Spec traceability below for why this
design already satisfies it, plus the one explicit response-code
decision it forces (un-favoriting a non-existent favorite now returns
404, not an idempotent success).

## Survey notes (Part 2)

`sandbox-svc-code` (confirmed code repo, `main` base branch) is a genuinely
greenfield repository: the root listing at `main` contains only `README.md`
("Dry-run sandbox code repo for nexai pipeline testing"). No `CLAUDE.md` or
`PROJECT.md`, no existing source tree, no shared library (e.g. a
`nexcommon`-style package) to survey for reuse. Stack choice is fully open
for this feature — there is no existing convention to respect or violate.

## Frontend

No UI scope in this pass. The frontend side is blocked on design/Figma and
is deferred to a later `/code-preparation`+`/fe-dev` pass once that unblocks.
`frontend_repo`/`frontend_base_branch` are intentionally omitted. This
session's design covers backend architecture only.

## Stack / Integration decisions

- **Stack: Go, using `nexcommon`** (user-confirmed; added as a new
  dependency to this greenfield repo — nothing to migrate, since there
  is no existing code). This is the team's standard shared library for
  Go backend services across this pipeline; a brand-new service with no
  conflicting convention is exactly the case where adopting it outright
  is the right default rather than a from-scratch stack.
- **Persistence: Postgres**, a single new table (`favorite_items`) via
  `nexcommon`'s data-access layer (DAO/repository pattern per
  `nexcommon-go-data-standards`). No separate datastore — this is a
  simple relation, not a reporting/analytics shape, so there's no
  reason to split it out from wherever `sandbox-svc-code`'s primary
  Postgres instance already is (or will be provisioned as part of this
  same greenfield setup).
- **Auth/session: consumed, not designed here.** Per the user: a shared
  auth middleware is assumed to exist by implementation time, placing
  an authenticated user ID into request context
  (`ContextModel`-style, per `nexcommon-go-standards`). This feature's
  handlers read that user ID; they do not implement login/session
  issuance. REQ-006 (anonymous users redirected to login) is satisfied
  by that middleware returning 401 before this feature's handlers ever
  run — the redirect itself is a frontend concern (out of scope this
  pass), triggered by the FE's handling of a 401 response.
- **Realtime "no reload" success criterion**: satisfied by client-side
  refetch-after-mutation (the FE re-queries `GET /favorites` after a
  successful favorite/un-favorite response). No push/websocket
  mechanism is designed server-side — user-confirmed default, revisit
  only if a later FE pass finds refetch-after-mutation insufficient in
  practice.

## Components

New components (all new — greenfield service, nothing "touched" vs.
"untouched" yet exists):

- **`favorite_endpoint`** (HTTP handlers): `POST /favorites`,
  `DELETE /favorites/{item_id}`, `GET /favorites`.
- **`favorite_service`**: business logic — idempotent add (REQ-005),
  the 500-cap check (REQ-007), remove, and the list-with-availability
  read path (REQ-004, joining current `catalog_items` status).
- **`favorite_dao`**: the `nexcommon`-pattern repository over
  `favorite_items` (Postgres).
- No new components in the existing catalog-items or auth/session
  systems — this feature only *reads* catalog-items state (to resolve
  current availability) and *reads* the auth-context user ID; it does
  not modify either.

**Backend authorization** (required regardless of UI scope, per the
guard): every endpoint above requires an authenticated user (enforced
by the assumed shared auth middleware, see above). There is no
role/permission distinction beyond "logged in" — a user may only ever
read/mutate their *own* favorites, scoped implicitly by the user ID
from request context, never by a user ID supplied in the
request body/query. There is no cross-user listing, no admin view, and
no separate "filter options" endpoint concern in this pass — REQ's
Priority section explicitly defers sorting/filtering of the favorites
list, so the filter-options-endpoint-per-filter principle (Part 5) has
nothing to apply to yet; it will apply when that later pass designs

**REQ-008 (row-ownership enforcement):** satisfied by construction,
not by an added check. `GET /favorites` always lists only the context
user's own rows, and `DELETE /favorites/{item_id}` always deletes by
`(context_user_id, item_id)` — there is no endpoint, in this design,
that addresses a favorite by any identifier other than the caller's
own `(user_id, item_id)` pair, so there is no code path where a
"mismatch" (this row belongs to someone else) is even distinguishable
from "this row doesn't exist": both cases are the exact same query
returning zero rows. The one behavior this REQ does pin down
explicitly: `favorite_service.Remove` must return **404 Not Found**
when `(context_user_id, item_id)` matches no row — previously
unspecified by REQ-003 alone — rather than treating un-favoriting a
non-favorited item as an idempotent no-op success (idempotency is
REQ-005's concern, and it's specific to *favoriting*, not
un-favoriting). FEAT-002's own per-note endpoints, once designed,
inherit this same pattern rather than re-deriving it.
filters.

## Data design

See `database.md` for the full schema-designer output (draft ERD,
normalization review, batched migration plan). Summary: one new table,
`favorite_items (user_id, item_id, created_at)`, composite primary key,
no soft-delete columns (un-favorite is a literal row delete, not a
lifecycle state), no surrogate key (nothing addresses a "favorite" by
any ID other than the (user, item) pair).

## Spec traceability

| Req | Requirement | Architectural answer |
|---|---|---|
| REQ-001 | Favorite from item detail page | `POST /favorites {item_id}` in `favorite_endpoint` -> `favorite_service.Add` -> `favorite_dao` INSERT. |
| REQ-002 | "My Favorites" page lists all favorited items | `GET /favorites` -> `favorite_service.List`, joins `favorite_items` to `catalog_items` for item details; FE renders without reload via its own refetch, not a backend concern. |
| REQ-003 | Un-favorite from detail page or list | `DELETE /favorites/{item_id}` -> `favorite_service.Remove` -> `favorite_dao` DELETE by `(user_id, item_id)` (user_id from auth context). Returns 404 if no matching row exists (see REQ-008). |
| REQ-004 | Discontinued favorited item still listed, marked unavailable | **Covered.** `favorite_service.List`'s join reads `catalog_items`' current availability/discontinued flag live at list-time (not denormalized at favorite-time). Confirmed correct per spec.md v2's clarification (resolving former CPF-001): the catalog-service team confirmed `catalog_items` rows are never hard-deleted — a discontinued/removed item is only ever soft-marked, the row itself always persists — so there is no hard-delete case where the row could silently disappear from the join. No schema change needed; the live-join design stands as originally proposed. |
| REQ-005 | Favoriting an already-favorited item is idempotent | `favorite_dao.Add` issues `INSERT ... ON CONFLICT (user_id, item_id) DO NOTHING`; the service layer treats both the newly-inserted and already-exists outcomes as the same success response. Enforced at the DB constraint level (composite PK), not just application logic. |
| REQ-006 | Anonymous users cannot favorite; redirected to login | Enforced by the assumed shared auth middleware (not this feature's code) returning 401 before any `favorite_endpoint` handler runs; the FE's redirect-on-401 behavior is explicitly out of scope this pass. |
| REQ-007 | Max 500 favorites; 501st attempt returns a clear error, no silent fail/evict | `favorite_service.Add` runs `COUNT(*) WHERE user_id = ?` (served by the PK's leading column, no extra index) before insert; at >= 500 it returns **HTTP 422 Unprocessable Entity** with a structured body `{"error_code": "FAVORITES_LIMIT_REACHED", "message": "..."}` and performs no insert and no eviction. (Spec says "a clear error" without naming a status code/shape — this is a technical decision made here, not a spec gap; see conversation.) |
| REQ-008 | Row-ownership enforcement on `favorite_items`; uniform 404 on mismatch-or-missing | **Satisfied by construction** — see the Components section's REQ-008 note. Every query is already scoped to `(context_user_id, item_id)`; a cross-user "mismatch" and a genuinely-missing row are the same zero-row result, so no separate ownership check is needed. The one new, explicit decision: `favorite_service.Remove` returns 404 (not an idempotent success) when no matching row exists. |
| Success criteria: immediate list update, no reload | — | Client-side refetch-after-mutation (see Stack/Integration decisions); no backend push mechanism. |
| Non-goal: sharing favorites | not applicable | No sharing endpoint/field exists; nothing to design against. |
| Non-goal: recommendation engine | not applicable | No recommendation logic touches this feature's components. |
| Non-goal/deferred: sort/filter on favorites list | not applicable, deferred | `GET /favorites` returns an unordered/insertion-order list this pass; sort/filter params and any filter-options endpoint are explicitly left to the later pass that un-blocks this. |
| Dependency: catalog items table/service | covered | Read-only join in `favorite_service.List`/`.Add` (to validate item exists and to resolve availability) — no write access, no new integration surface, in-process call (per spec's External Integrations note). |
| Dependency: auth/session system | covered | Consumed via request-context user ID from the assumed shared middleware; this feature designs no session/login logic of its own (user-confirmed). |

## Implementation Plan

1. **Data model and migration** — create `favorite_items` per
   `database.md`'s Batch 1 (and `nexcommon`/Postgres wiring if this is
   genuinely the first table in this greenfield repo).
2. **Core service logic** — `favorite_service`: idempotent add, 500-cap
   check, remove, list-with-availability join.
3. **API endpoints** — `favorite_endpoint`: wire the three HTTP routes
   to the service via `nexcommon`'s controller-wrapping conventions
   (`WrapService`-style), request/response DTOs, validation tags for
   `item_id`.
4. **Integration/observability wiring** — see Observability below;
   request metrics instrumentation for the new routes.
5. *(Frontend phase deliberately omitted — out of scope this pass; a
   later `/code-preparation` + `/fe-dev` pass adds it once design/Figma
   unblocks.)*

## Observability

- **Current state**: `sandbox-svc-code` is greenfield — no existing
  `/metrics` route, no pinned `nexcommon` version yet (this feature is
  the one adding `nexcommon` as a dependency in the first place).
- **Decision**: wire standard request metrics **once**, as part of
  this feature's implementation, following
  `nexcraft:nexcommon-go-prometheus-standards` for whichever
  `nexcommon` version gets pinned. There is nothing to "reuse and
  verify" yet since nothing instrumented exists.
- **Implementation boundary**: standard inbound HTTP request
  metrics (method/route/status, latency) on the three new
  `favorite_endpoint` routes, via `nexcommon`'s middleware/collector
  convention; `/metrics` exposed per that convention's default access
  rule. If the pinned `nexcommon` version has the known hardcoded-200
  status gap (per the guard's own warning), `/go-dev` must record
  either a verified library fix or status-aware service
  instrumentation — a status-blind wrapper alone is not acceptable,
  since REQ-007's 422 response is exactly the kind of non-200 outcome
  that would otherwise go unobserved.
- **Verification**: confirm a successful `POST /favorites` records a
  2xx-labeled metric and a REQ-007 cap-exceeded attempt records a
  422-labeled metric (not silently folded into 200), via a manual
  `/metrics` scrape during implementation.
- **Feature-specific metrics**: none needed beyond standard request
  metrics — there's no dashboard/counter requirement named by the spec
  or raised in discussion; inventing one would violate the guard's
  "do not invent dashboards" instruction.
