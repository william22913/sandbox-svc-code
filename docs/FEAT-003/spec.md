---
feature_code: FEAT-003
service: sandbox-svc
ticket_url: null
has_ui: false
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
pr_url: https://github.com/william22913/sandbox-docs/pull/17
last_updated: 2026-11-05
---

# FEAT-003 — Export my favorites to CSV (sandbox-svc)

## Problem

Employees who maintain favorites (FEAT-001) have no way to move that list
into a spreadsheet for monthly reporting. Today the only route is copying
rows by hand from the favorites list or querying the API row by row, which
does not scale and is error-prone. (source: user, this session)

## Target User

Any authenticated employee/user of `sandbox-svc`. The export is always run
by the user themselves, on their own favorites; a user with zero favorites
is still a valid caller (see REQ-011). (source: user, this session)

## Primary Flow

- **REQ-001**: A logged-in user calls `GET /favorites/export`; the service
  returns that user's current favorites as a downloadable CSV file.
- **REQ-002**: A successful response is `200` with `Content-Type: text/csv`
  and `Content-Disposition: attachment;
  filename="favorites-<user_id>-<YYYY-MM-DD>.csv"`, where the date is the
  export date in UTC.
- **REQ-003**: The file contains one header row followed by one data row per
  current favorite. There is no paging and no filter: every one of the
  user's favorites is included (already bounded by FEAT-001's 500-per-user
  cap), ordered newest-favorited first, matching `GET /favorites` order.
- **REQ-004**: The header row is exactly
  `item_id,name,category,price,available,favorited_at,note`, in that column
  order.

## Data Validation Rules

These are the rules the exported file must satisfy. FEAT-003 accepts no
business input fields, so none of these constrain a request payload.

- **REQ-005**: `price` is written as a raw number, with no currency symbol,
  no thousands separator, and no rounding applied.
- **REQ-006**: `favorited_at` is written as an ISO date `YYYY-MM-DD` in UTC.
- **REQ-007**: `available` is written as `Tersedia` when the item is
  available and `Tidak Tersedia` when it is discontinued/unavailable,
  matching the labels FEAT-001's list view uses.
- **REQ-008**: `note` is written verbatim, as already trimmed and
  length-validated by FEAT-002. A favorite with no note, or an empty note,
  exports as an empty cell.
- **REQ-009**: The file is UTF-8 **with BOM**, uses CRLF line endings and a
  comma delimiter, and quotes every field containing a comma, double quote,
  CR, or LF, doubling embedded double quotes (RFC 4180).
- **REQ-010**: Any value in `name`, `category`, or `note` that begins with
  `=`, `+`, `-`, or `@` is neutralized so a spreadsheet opening the file
  cannot execute it as a formula (CSV formula injection).

(source for REQ-005–REQ-010: user, this session)

## Edge Cases

- **REQ-011**: A user with zero favorites receives `200` and a CSV
  containing only the header row — not an error, and not an empty body.
- **REQ-012**: An unauthenticated caller receives the existing `401`
  `UNAUTHENTICATED` error envelope and no file.
- **REQ-013**: The export is scoped only to the authenticated user's own
  rows. The user id comes from the auth context; no request parameter or
  body field can select another user. There is no admin or cross-user
  export.
- **REQ-014**: A favorited item that is now discontinued/unavailable still
  appears in the export (per FEAT-001's REQ-004) with `available` =
  `Tidak Tersedia`; it is never silently dropped.
- **REQ-015**: Values are read live from `favorite_items` → `catalog_items`
  at export time; the export does not snapshot or cache item fields, so a
  renamed, repriced, or discontinued item exports its current state.

(source for REQ-011–REQ-015: user, this session)

## Dependencies

- **REQ-016**: Requires FEAT-001's `favorite_items` table, its live join to
  `catalog_items` (for `name`, `category`, `price`, and availability), and
  FEAT-001's REQ-008 row-ownership rule — which FEAT-003 satisfies the same
  way, by scoping every read to the context user id and accepting no
  caller-supplied user identifier.
- **REQ-017**: Requires FEAT-002's `note` field on `favorite_items` for the
  `note` column. A row with no note exports an empty cell (REQ-008).
- **REQ-018**: Requires the shared auth middleware assumed by FEAT-001 to
  place the authenticated user id in request context and to return `401`
  for anonymous callers. FEAT-003 designs no login/session logic of its
  own.

## Non-Goals

- No UI. Triggering the export is backend/API only. (source: user, this
  session)
- No month or date-range filtering — "monthly" is the reporting cadence,
  not a filter on the export.
- No formats other than CSV (no XLSX, no JSON, no PDF).
- No scheduled, automated, or emailed exports; the user triggers each one.
- No admin or cross-user export.
- No export history or audit log.
- No writes to favorites or catalog data — the export is strictly
  read-only.

## Priority

This pass covers the whole feature: the single synchronous
`GET /favorites/export` endpoint, its file format rules, and the
authentication/ownership behavior (REQ-001–REQ-021).

Explicitly deferred to a later pass:
- Month/date-range filtering.
- Additional export formats (XLSX, JSON).
- Scheduled or emailed reports.
- Admin/cross-user export.
- Export history/audit log.
- Any UI trigger.

(source: user, this session — nothing deferred is needed for the first
monthly run)

## Success Criteria

- **REQ-019**: For a logged-in user with N favorites, the download is a CSV
  with exactly 1 header row and N data rows, one per current favorite,
  newest-first, whose values match what `GET /favorites` returns (including
  note and availability). (verifies REQ-001–REQ-008)
- **REQ-020**: A user with zero favorites gets a header-only CSV with
  `200`. (verifies REQ-011)
- **REQ-021**: No request input can cause the export to return another
  user's rows. (verifies REQ-013)
- No post-launch business metric — reporting is ad hoc, so there is no
  dedicated analytics requirement beyond the acceptance criteria above.
  (source: user, this session)

## External Integrations

None, because the catalog-items data and the auth/session system are
internal in-process packages within the same `sandbox-svc` monolith — same
as FEAT-001 — so there is no network-exposed external integration to
specify a contract for. (source: user, this session)
