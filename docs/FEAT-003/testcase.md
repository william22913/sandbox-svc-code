---
feature_code: FEAT-003
service: sandbox-svc
version: 1
status: complete
spec_version: 1
open_questions: []
pr_url: https://github.com/william22913/sandbox-docs/pull/18
last_updated: 2026-11-05
---

# FEAT-003 — Export my favorites to CSV — Test Cases

Black-box test design for the single synchronous `GET /favorites/export`
endpoint. Cases describe behavior observable at the API/file layer; none
has been executed by this document.

## Requirement coverage

| Requirement | Description | Test cases | Notes |
|---|---|---|---|
| REQ-001 | Logged-in user gets their own favorites as a downloadable CSV | TC-001 | |
| REQ-002 | `200`, `text/csv`, attachment filename `favorites-<user_id>-<YYYY-MM-DD>.csv` with UTC export date | TC-001, TC-002, TC-003 | |
| REQ-003 | Header + one row per current favorite; no paging/filter; newest-favorited first, matching `GET /favorites` | TC-005, TC-006, TC-007, TC-008, TC-009 | |
| REQ-004 | Header row exactly `item_id,name,category,price,available,favorited_at,note` | TC-004 | |
| REQ-005 | `price` raw number: no symbol, no thousands separator, no rounding | TC-010, TC-011, TC-012 | |
| REQ-006 | `favorited_at` as ISO `YYYY-MM-DD` in UTC | TC-013, TC-014 | |
| REQ-007 | `available` as `Tersedia` / `Tidak Tersedia` | TC-015, TC-016 | |
| REQ-008 | `note` verbatim; absent or empty note exports as an empty cell | TC-017, TC-018, TC-019 | |
| REQ-009 | UTF-8 with BOM, CRLF, comma delimiter, RFC 4180 quoting and quote doubling | TC-020, TC-021, TC-022, TC-023, TC-024 | |
| REQ-010 | Leading `=`, `+`, `-`, `@` in `name`/`category`/`note` neutralized | TC-025, TC-026 | |
| REQ-011 | Zero favorites: `200` header-only CSV, not an error or empty body | TC-027 | |
| REQ-012 | Unauthenticated caller: existing `401` `UNAUTHENTICATED` envelope, no file | TC-028, TC-029 | |
| REQ-013 | Export scoped to auth-context user; no parameter/body can select another user | TC-030, TC-031 | |
| REQ-014 | Discontinued/unavailable favorited item still exported, labelled `Tidak Tersedia` | TC-016 | |
| REQ-015 | Values read live from `favorite_items` → `catalog_items` at export time | TC-032 | |
| REQ-016 | Dependency: FEAT-001 table, live join, and row-ownership rule | TC-030, TC-031 | A dependency requirement, not a separate behavior: verified through the ownership cases it mandates. |
| REQ-017 | Dependency: FEAT-002 `note` field on `favorite_items` | TC-017, TC-018, TC-019 | A dependency requirement; its behavior is the `note` column cases. |
| REQ-018 | Dependency: shared auth middleware places user id in context and returns `401` for anonymous callers | TC-028 | A dependency requirement; verified through the unauthenticated case. |
| REQ-019 | Success criterion: N favorites → 1 header + N rows, newest-first, values match `GET /favorites` | TC-005 | |
| REQ-020 | Success criterion: zero favorites → header-only CSV with `200` | TC-027 | |
| REQ-021 | Success criterion: no request input can return another user's rows | TC-030, TC-031 | |

Non-goals (no UI trigger, no date-range filtering, no other formats, no
scheduled/emailed exports, no admin/cross-user export, no export history,
no writes) have no corresponding feature behavior to exercise; they are
deliberately not turned into pass/fail cases except where the spec
implies rejection behavior (REQ-013/REQ-021, covered by TC-030/TC-031).

## Test cases

| ID | Type | Spec ref (REQ-*) | Preconditions / test data | Action / input | Expected result | Priority | Level | Automation candidate |
|---|---|---|---|---|---|---|---|---|
| TC-001 | Positive | REQ-001, REQ-002 | User U is authenticated with a valid session and has at least one favorite in `favorite_items`. | U calls `GET /favorites/export` with no body. | Response is `200`, `Content-Type` is `text/csv`, and a file download is returned (not JSON). File is non-empty and begins with the header required by REQ-004. | High | API | Yes |
| TC-002 | Positive | REQ-002 | User U is authenticated; U's auth-context user id is known; export date is `2026-11-05` in UTC. | U calls `GET /favorites/export`. | `Content-Disposition` is `attachment; filename="favorites-<U's user id>-2026-11-05.csv"` — U's own id from the auth context and the UTC export date, exactly in that shape. | High | API | Yes |
| TC-003 | Boundary | REQ-002 | User U is authenticated; server's local time zone is not UTC (e.g. Asia/Jakarta, UTC+7). | Two exports: one at `2026-11-05T23:59:59Z`, one at `2026-11-06T00:00:01Z`. | First filename contains date `2026-11-05`; second contains `2026-11-06`. The date is the UTC export date, not the server-local date (which would be `2026-11-06` and `2026-11-06` in UTC+7). | High | API | Yes |
| TC-004 | Positive | REQ-004 | User U is authenticated and has at least one favorite. | U calls `GET /favorites/export` and reads the first line. | The header row is exactly `item_id,name,category,price,available,favorited_at,note` — exact spelling, lower-case, and column order, with no extra columns and no BOM characters inside the first field. | High | API | Yes |
| TC-005 | Positive | REQ-003, REQ-019 | User U has exactly 3 favorites: A (oldest), B (middle), C (newest), with distinct `favorited_at` timestamps and different `name`/`category`/`price`/availability/note values. | U calls `GET /favorites/export`; then U calls `GET /favorites`. | File has exactly 1 header row + 3 data rows, in order C, B, A (newest-favorited first). Every field of every row matches the corresponding value returned by `GET /favorites`, including `note` and `available`. | High | API | Yes |
| TC-006 | Boundary | REQ-003 | User U has exactly 500 favorites — FEAT-001's per-user cap — all with distinct `favorited_at` values. | U calls `GET /favorites/export`. | File has exactly 1 header row + 500 data rows; no row is dropped, truncated, or paged out. | High | API | Yes |
| TC-007 | Boundary | REQ-003 | User U has exactly 1 favorite. | U calls `GET /favorites/export`. | File has exactly 1 header row + 1 data row (smallest non-empty export); the single row's values are correct. | Medium | API | Yes |
| TC-008 | Boundary | REQ-003 | User U has two favorites X and Y whose `favorited_at` values are identical (same instant). | U calls `GET /favorites/export`; compare with U's `GET /favorites`. | The two rows appear in the same relative order as `GET /favorites` returns them; the export does not invent a different tie order. | Medium | API | Yes |
| TC-009 | Negative | REQ-003 | User U has N > 0 favorites, spanning at least two categories. | U calls `GET /favorites/export?page=2&category=<one category>`. | The response still contains all N favorites with 1 header + N data rows; the request is not paginated or filtered into a subset, because the endpoint supports neither. | Medium | API | Yes |
| TC-010 | Positive / Equivalence partition (integer) | REQ-005 | User U has one favorite whose catalog `price` is the integer `1500`. | U calls `GET /favorites/export` and extracts the `price` cell. | The cell holds the numeric value 1500; it contains no currency symbol (no `Rp`, `$`, etc.) and no thousands separator (`1,500` is forbidden). | High | API | Yes |
| TC-011 | Positive / Equivalence partition (fractional) | REQ-005 | User U has one favorite whose catalog `price` is `19.99`. | U calls `GET /favorites/export` and extracts the `price` cell. | The cell parses as exactly `19.99` — not `20`, not `19.990000`, and not a rounded/abbreviated form; no currency symbol or thousands separator. | High | API | Yes |
| TC-012 | Boundary (precision) | REQ-005 | User U has one favorite whose catalog `price` is a value stored exactly by the schema, e.g. `1234.5678`. | U calls `GET /favorites/export` and extracts the `price` cell. | The cell parses as exactly `1234.5678`; the export applies no rounding and no thousands separator (`1,234.5678` is forbidden). | Medium | API | Yes |
| TC-013 | Positive | REQ-006 | User U has a favorite whose `favorited_at` is `2026-04-09T14:30:00Z`. | U calls `GET /favorites/export` and extracts the `favorited_at` cell. | The cell is exactly `2026-04-09` — ISO `YYYY-MM-DD`, zero-padded, no time component and no time-zone suffix. | High | API | Yes |
| TC-014 | Boundary | REQ-006 | User U has two favorites: X favorited at `2026-02-28T23:59:59Z`, Y favorited at `2026-03-01T00:00:00Z`. | U calls `GET /favorites/export`. | X's `favorited_at` cell is `2026-02-28`; Y's is `2026-03-01`. The UTC-midnight boundary is not shifted by the server's local time zone. | High | API | Yes |
| TC-015 | Positive / Equivalence partition (available) | REQ-007 | User U has a favorite for an item whose `catalog_items` availability is true (not discontinued). | U calls `GET /favorites/export`. | The row's `available` cell is exactly `Tersedia`. | High | API | Yes |
| TC-016 | Positive / Equivalence partition (unavailable) | REQ-007, REQ-014 | User U favorited item X while it was available; X is now marked discontinued/unavailable in `catalog_items`. | U calls `GET /favorites/export`. | Item X still appears as a data row (not silently dropped) and its `available` cell is exactly `Tidak Tersedia`. | High | API | Yes |
| TC-017 | Positive | REQ-008, REQ-017 | User U has a favorite with no note row/field value at all (never set since FEAT-002). | U calls `GET /favorites/export`. | The row's `note` cell is empty — the delimiter immediately follows the `favorited_at` value and the row ends after the empty `note` field. No placeholder such as `null`, `None`, or `-`. | High | API | Yes |
| TC-018 | Positive | REQ-008, REQ-017 | User U has a favorite whose stored note is the empty string `""`. | U calls `GET /favorites/export`. | The row's `note` cell is empty, identical to TC-017; an empty note is not rendered as `""` (two quote characters) or as a literal placeholder. | Medium | API | Yes |
| TC-019 | Positive | REQ-008, REQ-017 | User U has a favorite whose stored note is `Café — 🎁 for mom` (mixed accented Latin, em dash, emoji, internal spaces). | U calls `GET /favorites/export`. | The `note` cell contains exactly `Café — 🎁 for mom`, byte-for-byte after RFC 4180 unquoting: no trimming, normalization, transliteration, or character replacement by the export. | High | API | Yes |
| TC-020 | Positive | REQ-009 | User U has at least one favorite. | U downloads the file and inspects its raw bytes and line terminators. | Byte 0–2 are the UTF-8 BOM `EF BB BF` (present exactly once). Every record terminator is CRLF (`0D 0A`); no lone `0A` or `0D` separates records outside a quoted field. Fields are comma-delimited. | High | API | Yes |
| TC-021 | Positive | REQ-009 | User U has a favorite whose item name is `Notebook, A5` (contains a comma). | U calls `GET /favorites/export` and reads that row's raw text. | The `name` field is double-quoted: `"Notebook, A5"`. It is not split into two fields, and the row still has exactly 7 fields. | High | API | Yes |
| TC-022 | Positive | REQ-009 | User U has a favorite whose item name is `He said "hi", twice` (contains double quotes and a comma). | U calls `GET /favorites/export` and reads that row's raw text. | The field is quoted and every embedded double quote is doubled: `"He said ""hi"", twice"`. Unquoting it yields the original string exactly. | High | API | Yes |
| TC-023 | Positive | REQ-009 | User U has a favorite whose note contains an embedded CR and LF, e.g. `line1\r\nline2`. | U calls `GET /favorites/export` and reads that row's raw text. | The `note` field is double-quoted and the embedded CR and LF are preserved verbatim inside the quotes; the embedded CRLF is not treated as the end of the record. | Medium | API | Yes |
| TC-024 | Positive | REQ-009 | User U has a favorite whose `name` is `Café`, `category` is `Bücher`, and `note` is `日本語`. | U calls `GET /favorites/export` and decodes the file as UTF-8 with BOM. | The file decodes cleanly as UTF-8; the three cells read exactly `Café`, `Bücher`, and `日本語` — no mojibake, replacement characters, or HTML/entity escaping. | Medium | API | Yes |
| TC-025 | Positive / Security | REQ-010 | User U has 12 favorites covering the full field × character matrix: for each of the fields `name`, `category`, and `note`, one favorite whose value in that field begins with each of `=`, `+`, `-`, and `@` (e.g. `=SUM(A1:A9)`, `+1+1`, `-1+1`, `@SUM(1+1)`); the other columns hold ordinary values. | U calls `GET /favorites/export`, unquotes every field, and opens the file in a spreadsheet application. | For all 12 exported cells, the value in that field does not begin with `=`, `+`, `-`, or `@` — it has been neutralized. Opening the file displays the original text literally; the spreadsheet does not evaluate a formula, external command, or link for any of the 12 values. | High | API | Yes |
| TC-026 | Boundary | REQ-010 | User U has two favorites: item A's name is ` =SUM(A1:A9)` (one leading space before `=`); item B's name is `Total = 5` (the `=` is mid-string). | U calls `GET /favorites/export` and reads both `name` cells. | Neither cell is neutralized: only a value that *begins* with `=`, `+`, `-`, or `@` is in scope. Both cells unquote to exactly the original text, with no added prefix or altered characters. | Medium | API | Yes |
| TC-027 | Positive | REQ-011, REQ-020 | User U is authenticated and has zero favorites. | U calls `GET /favorites/export`. | Response is `200` (not `404`, `204`, or an error envelope). The body is non-empty and consists of the BOM + the exact header row + CRLF, with 1 header row and 0 data rows. `Content-Type: text/csv` and an attachment `Content-Disposition` are still present. | High | API | Yes |
| TC-028 | Negative | REQ-012, REQ-018 | No authenticated session: the request carries no credentials/token. | Caller calls `GET /favorites/export` anonymously. | Response is `401` with the existing shared `UNAUTHENTICATED` error envelope. No CSV body, no `text/csv` content type, and no attachment `Content-Disposition` are returned. | High | API | Yes |
| TC-029 | Negative | REQ-012, REQ-018 | Caller presents a malformed, expired, or otherwise invalid token. | Caller calls `GET /favorites/export`. | Response is `401` with the same `UNAUTHENTICATED` envelope as TC-028; no CSV file is returned. | Medium | API | Yes |
| TC-030 | Negative / Authorization | REQ-013, REQ-016, REQ-021 | User A and user B both have favorites. A is authenticated; B's user id is known to the test but A supplies it in the request. | A calls `GET /favorites/export?user_id=<B>&userId=<B>` and separately `GET /favorites/export` with header `X-User-Id: <B>`. | Both responses are `200` and contain only A's rows. None of B's items appear. The caller-supplied id is ignored because the export uses only the auth-context user id; no parameter or header can select another user. | High | API | Yes |
| TC-031 | Positive / Authorization | REQ-013, REQ-016, REQ-021 | User A and user B are each authenticated in separate sessions, with disjoint favorites. | A and B each call `GET /favorites/export`. | A's file contains exactly A's favorites and B's file contains exactly B's favorites; neither file contains a row belonging to the other user, and row counts match each user's `GET /favorites`. | High | API | Yes |
| TC-032 | Positive | REQ-015 | User U favorited item X when X's name was `Old name`, price was `10`, and it was available. Before the export, `catalog_items` for X is updated to name `New name`, price `12.5`, and unavailable. | U calls `GET /favorites/export`. | Item X's row shows `New name`, the numeric price `12.5`, and `Tidak Tersedia` — the live `catalog_items` state at export time, not a snapshot taken when X was favorited. The row's `favorited_at` still reflects when X was favorited. | High | API | Yes |

## Open questions for /document-preparation

None. Every expected result above follows from spec.md v1; where the
spec fixes a security property rather than an exact escaping character
(REQ-010), the case asserts the property (the cell must not begin with
`=`, `+`, `-`, or `@` and must render as literal text) rather than
inventing a specific escape character.
