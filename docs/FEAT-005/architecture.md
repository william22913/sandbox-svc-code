---
feature_code: FEAT-005
service: sandbox-svc
version: 1
status: complete
spec_version: 1
code_repo: https://github.com/william22913/sandbox-svc-code
release_branch: main
integration_branch: feat/FEAT-005
stack: go
frontend_repo: https://github.com/william22913/sandbox-svc-ui
frontend_base_branch: main
dev_feedback: []
code_prep_feedback: []
sources: []
pr_url: https://github.com/william22913/sandbox-docs/pull/25
last_updated: 2026-11-08
---

# FEAT-005 — Favorites summary and badge tooltip (sandbox-svc) — Architecture

## Revision notes (spec v1)

First design pass for this feature. `spec.md` v1 is complete and merged
(PR 23); `testcase.md` v1 is complete and merged (PR 24) and pins
`spec_version: 1`. All 29 requirements are covered below.

The feature is small and read-only: one aggregate endpoint and a hover
tooltip on FEAT-004's existing badge. It adds **no schema change**
(REQ-019). The design work is concentrated in four places:

1. **One aggregate query, no cache.** `total` and `discontinued` are
   derived in a single indexed `SELECT` over the caller's
   `favorite_items` rows joined to `catalog_items`. Nothing is stored,
   cached, or denormalized (REQ-009), and the spec's Non-Goals forbid a
   counter column or cache.
2. **Reuse FEAT-004's frontend provider rather than build a second
   one.** FEAT-004 already owns the badge's count state, its visibility
   rules, and the `refresh()` trigger after a favorite/un-favorite.
   FEAT-005 extends that provider with summary state and adds the
   tooltip; it does not add a second badge, a second provider, or a
   second refresh path (REQ-021, REQ-023).
3. **First-hover fetch, explicit invalidation.** The summary is fetched
   on the first hover of a visible badge, not on app load (REQ-007).
   After a successful favorite/un-favorite an already-fetched summary is
   thrown away before it is re-fetched, so pre-action values can never be
   shown (REQ-018).
4. **The tooltip is display-only and hover-only.** It renders only while
   the badge is visible and the summary is ready; there is no click
   action, focus trigger, spinner or error UI (REQ-005, REQ-006,
   REQ-008, REQ-015).

Two source-repo preconditions are recorded as documented deviations
(see the last section): the backend code baseline is not `release_branch`
yet, and FEAT-004's frontend badge is designed but not yet implemented.

## Survey notes (Part 2)

Both repositories were surveyed read-only on the git host. Neither was
cloned or modified.

**`sandbox-svc-code`** — surveyed at
`feat/FEAT-003-dev-export-favorites-csv`, the newest ref that actually
contains the real code. The branch topology matters here and is stated
plainly rather than assumed:

- `main` holds only the repository `README.md`.
- `feat/FEAT-001` is merged (PR 1) and carries the favorite/un-favorite/
  list implementation.
- `feat/FEAT-003` carries FEAT-001 plus FEAT-003's docs, but not
  FEAT-003's implementation.
- `feat/FEAT-003-dev-export-favorites-csv` carries the real FEAT-003 CSV
  implementation, and has **PR 2 open** against `feat/FEAT-003` — not
  merged.
- `feat/FEAT-004` carries only FEAT-004's materialized design docs; its
  count endpoint is not implemented in code yet.

This design reads the real code from
`feat/FEAT-003-dev-export-favorites-csv`, per the user's instruction to
survey the latest branch with the real code (FEAT-003 or newer). What
that survey established, and what this design reuses rather than
invents:

- **Stack** — Go 1.27, `gorilla/mux`, Postgres via `database/sql`+pgx,
  `sql-migrate`, `nexcommon v0.3.68-swagger`.
- **Route prefix** — `router.V1PrefixPath = "/v1"`;
  `favorite_endpoint.Path = router.V1PrefixPath + "/favorites"`. The real
  paths are `/v1/...`; FEAT-001's architecture.md writes them unprefixed
  as shorthand. This document uses the real paths only.
- **Existing routes** — `POST /v1/favorites` (`registerAdd`),
  `GET /v1/favorites` (`registerList`), `DELETE
  /v1/favorites/{item_id}` (`registerRemove`), plus FEAT-003's raw
  `GET /v1/favorites/export`, `/v1/health` and `/metrics` registered in
  `router.InitHttpService`.
- **Auth** — `httpValidator.UserAccessValidator()` over Redis. A
  missing/unknown token returns
  `errors.ErrUnauthorized` (`401`, code `E-1-CMD-AUT-001`) before any
  service method runs. On success it populates
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`.
- **Error envelope (verified against the vendored library itself,
  `vendor/github.com/nexsoft-git/nexcommon/dto/out/out.go`, not
  assumed)** — `{"success": false, "header": {"request_id", "version",
  "timestamp"}, "payload": {"status", "code", "message"}}`. The type is
  `out.DefaultErrorResponse`; there is no top-level `error_code`/`message`
  pair in the wrapped controller's wire format. Unknown errors become
  `500` with `E-5-SBX-SRV-001`.
- **Success envelope** — `out.DefaultResponse` emits `{"success": true,
  "header": {...}, "payload": {...}}`; `DTOOut` shapes `payload` only.
- **Existing DAO** — `dao.FavoriteDAO` exposes `Add`, `Remove`, `List`,
  `ListForExport` and `ItemExists` over `*sql.DB`, each scoped by
  `user_id`; it returns raw errors (rendered as a generic `500`) and uses
  return values for business conditions.
- **Existing service** — `favorite.Service` methods have the signature
  `func (s *Service) X(ctx *internalCtx.ContextModel, _ model.URLParam,
  _ interface{}) (map[string]string, interface{}, error)`, returning
  `(nil, <dto>, nil)` on success.
- **Existing schema** — `favorite_items(user_id, item_id, created_at,
  note)` with composite PK `(user_id, item_id)`; `catalog_items(id,
  name, is_available, created_at, category, price)`. A favorite's
  "discontinued/unavailable" state is `catalog_items.is_available =
  false`, read live at request time (`dao.List` uses `LEFT JOIN
  catalog_items ... COALESCE(c.is_available, FALSE)`).
- **Metrics** — `middleware.StatusAwareMetricsMiddleware` is installed
  with `router.Use(metricMiddleware.Serve)` and reads the real response
  status; `/metrics` is served outside token validation. FEAT-003's live
  run confirms status-labelled samples.

**`sandbox-svc-ui`** — surveyed at `main`, which now contains FEAT-001's
merged app (PR 1 merged). What the FE survey established:

- **Stack** — React 18.2 + Vite 5 + JSX, Ant Design 4.24,
  `nexcomponent-ui` 2.1.9 vendored as
  `file:vendor/nexcomponent-ui-2.1.9.tgz`, axios 1.7, react-router-dom
  6.30, i18next 23 (Indonesian), vitest 2, Playwright 1.64.
- **Shell** — `src/app/AppShell.jsx` renders a sidebar (`shell__sidebar`
  with `shell__brand` + `shell__nav`) and a main region with `<Outlet />`.
  Every signed-in route in `src/app/AppRoutes.jsx` is nested under it;
  `/login` is outside. **There is still no `<header>` element.**
- **API client** — `src/core/api/axios/client.js` sets `baseURL:
  VITE_API_BASE_URL || '/api'`, `withCredentials: true`, and redirects to
  `/login` on any `401`.
- **Favorites data flow** — `favoriteAdapter.js` → `favoriteService.js`
  → `useFavoriteStar` / `useFavorites`. `useFavoriteStar.toggle` and
  `useFavorites.remove` are the two success paths REQ-023 names.
- **Deployment** — `nginx.conf` serves the SPA and rewrites
  `location /api/` to the backend root, stripping `/api`. The Vite dev
  proxy forwards `/api` to `localhost:8080` **without** a rewrite.
- **FEAT-004 is designed but not implemented.** There is no
  `FavoriteCountProvider`, no `useFavoriteCount`, no badge in
  `AppShell`, and no count adapter/service in the surveyed tree. FEAT-005
  therefore extends FEAT-004's design rather than existing code; see
  deviations 3 and 4.

## Frontend

**UI scope: yes** (`has_ui: true`). `frontend_repo` and
`frontend_base_branch` are set above. There is no `UIMap.md` for this
feature — `/document-preparation` produced `PRD.md` and `AppFlow.md`
only — so this section records routes, components, states and behaviour
at the level the requirements justify. Visual detail (tooltip styling,
delay, placement) is a `/ui-preparation` concern if the team wants one
later; nothing below depends on it.

**Precondition — FEAT-004's badge must exist first.** FEAT-005 adds
hover behaviour to FEAT-004's badge and reuses FEAT-004's
`FavoriteCountProvider` (`useFavoriteCount()` with `count` and
`refresh()`). Neither exists in the surveyed FE tree: FEAT-004 has a
merged design but no implementation. `/fe-dev` must either run after
FEAT-004's frontend work has landed in the FE repo, or record a
`dev_feedback` entry rather than silently building a second badge or a
second provider. This is a sequencing precondition, not a design change.

**Routes.** No new route. The tooltip attaches to the badge that FEAT-004
renders inside `AppShell`, so it is available on every signed-in screen
that already exists: `/favorites`, `/catalog`, `/catalog/:itemId`, and
the index redirect to `/favorites`. `/login` is outside the shell, so an
anonymous visitor never renders the provider, the badge, or the
tooltip.

**Components.**

- `src/features/favorites/FavoriteCountProvider.jsx` (FEAT-004, changed)
  — extend the provider, do not replace it. FEAT-004 already exposes
  `{ count, refresh }`. Add:
  - `summary`: `{ total, discontinued } | null` — never shown until a
    successful fetch has produced both integers.
  - `summaryStatus`: `'idle' | 'loading' | 'ready' | 'error'`. `'idle'`
    means "never fetched" (the state that must stay first-hover-driven
    per REQ-007).
  - `ensureSummary()` — called on hover. No-op when the status is
    `'loading'` or `'ready'`; fetches when `'idle'` or `'error'` (so a
    later hover retries after a failure, REQ-008). On success it
    validates both values are finite, non-negative integers with
    `discontinued <= total`, then sets `'ready'`; any malformed response
    is treated as `'error'` (no tooltip), never as zeros.
  - `refreshSummary()` — the post-action path (REQ-018, REQ-023). If the
    summary was never fetched (`'idle'`), this is a no-op. Otherwise it
    synchronously drops the held values (`summary = null`,
    `'loading'`) and re-fetches. A failed refresh leaves `'error'` with
    no values; the pre-action values are gone and can never be rendered.
  - The tooltip text is built by one pure helper in this module,
    `formatFavoriteSummary({ total, discontinued })`, returning the
    literal `` `${total} favorites (${discontinued} discontinued)` ``.
    No pluralization and no i18n key — the spec's Non-Goals require the
    literal template, including `1 favorites` and `(0 discontinued)`.
- `src/app/AppShell.jsx` (FEAT-004, changed) — wrap FEAT-004's badge
  `<span>` in Ant Design's `Tooltip`; the badge's own text, `99+` cap,
  visibility rule and styles are unchanged:

  ```jsx
  const [hovered, setHovered] = useState(false);

  <Tooltip
    open={hovered && summary !== null}
    title={summary ? formatFavoriteSummary(summary) : undefined}
    onOpenChange={(open) => {
      setHovered(open);
      if (open) ensureSummary();
    }}
  >
    <span className="shell__badge" aria-label={...}>{display}</span>
  </Tooltip>
  ```

  `open` is controlled so that no tooltip can appear while the summary
  is idle, loading or failed — the title alone being set is not enough.
  Because the badge is only rendered when FEAT-004's count is a positive
  number, a hidden badge has no trigger at all: no tooltip and no fetch
  (REQ-006, REQ-015). The badge stays a display-only `<span>` with no
  click handler and no navigation (REQ-005, spec Non-Goals).
- `src/core/api/adapters/favoriteAdapter.js` (changed) — add
  `getFavoritesSummary()` calling `client.get('/v1/favorites/summary')`.
  The path **does** carry `/v1`: nginx strips only `/api`, so
  `/api/v1/favorites/summary` reaches the backend as
  `/v1/favorites/summary`.
- `src/core/services/favoriteService.js` (changed) — add
  `fetchFavoritesSummary()` returning
  `{ total: Number(envelope?.payload?.total), discontinued:
  Number(envelope?.payload?.discontinued) }`, or `null` when either
  value is not a finite number. It never coerces a malformed response to
  `{0,0}`.
- `src/features/favorites/hooks/useFavoriteStar.js` (FEAT-004, changed)
  — in the success branch of `toggle`, call `refreshSummary()` next to
  FEAT-004's existing `refresh()`. The optimistic flip and rollback
  behaviour is unchanged; only the success branch gains the call.
- `src/features/favorites/hooks/useFavorites.js` (FEAT-004, changed) —
  after a successful `remove`, call `refreshSummary()` next to
  `refresh()`.
- `src/locales/id.json` — **no change.** The tooltip text is a literal
  English template required by the spec; it is not user-interface chrome
  covered by the app's Indonesian locale.
- No CSS change is required beyond whatever FEAT-004 adds for the badge;
  Ant Design's tooltip supplies its own overlay.

**State and display rules** (the table the tests exercise):

| State | Badge | Tooltip | Fetch |
|---|---|---|---|
| Count `null` (loading/error) or `0`, i.e. badge hidden | hidden (FEAT-004) | none — no trigger exists | none (REQ-006, REQ-015) |
| Badge visible, summary `'idle'` | unchanged | none yet; hover starts the one fetch | first hover only (REQ-007) |
| Summary `'loading'` | unchanged | none | in flight |
| Summary `'ready'` | unchanged | `N favorites (M discontinued)`, exact integers | repeat hover does not refetch (REQ-007) |
| Summary `'error'` | unchanged | none; a later hover retries | retry on hover (REQ-008) |
| After a successful favorite/un-favorite | FEAT-004 refreshes its count as usual | none until the summary refresh completes; pre-action values never shown | only if a summary had been fetched (REQ-018, REQ-023) |

The `99+` cap applies to the badge only; `formatFavoriteSummary` always
renders the exact `total` (REQ-005, REQ-027).

**Accessibility and input modality.** The badge keeps FEAT-004's
`aria-label`; Ant Design's tooltip adds the visual hover overlay. Hover
is the only specified trigger (spec Non-Goals), so no focus/click
trigger is added and no keyboard-specific behaviour is invented. Touch
devices have no hover, and the spec specifies none; the feature is
desktop-hover-only by requirement, not by omission.

**Tests.** vitest for the pure formatter (`0/0`, `1/0`, `3/1`, `3/3`,
`100/12`, `150/3`) and the service mapper (valid envelope, missing
`payload`, non-numeric, `discontinued > total`); provider tests for
mount-without-fetch, first-hover fetch, repeat hover no-refetch, error
then retry, post-action invalidation, and never-fetched no-op. Playwright
smoke: mock `GET /api/v1/favorites/summary` with the real
`payload.total`/`payload.discontinued` envelope and assert no request on
load, the exact tooltip text on hover, no tooltip while loading or after
a `500`, and refreshed values after a mocked favorite action.

**Deployment.** No new dependency and no Docker change. The Vite dev
proxy needs the same `/api` rewrite FEAT-004 specifies
(`rewrite: (path) => path.replace(/^\/api/, '')`) for `/v1` paths to be
reachable in `npm run dev`; if FEAT-004's change is already in the base,
reuse it. Production behaviour is unchanged.

## Stack / Integration decisions

- **Stack: Go, `nexcommon`** — unchanged from FEAT-001/FEAT-003/FEAT-004;
  this feature adds no dependency and no new library. `stack: go`, so the
  next command is `/go-dev`.
- **Persistence: Postgres, reuse FEAT-001's `favorite_items` and the
  existing `catalog_items`.** One aggregate query per request; no join
  beyond the live availability lookup the list already performs, no
  cache, no denormalized counter. The spec's Non-Goals explicitly forbid
  a cache, a new table, a new column and a new index.
- **Auth/session: reused, not assumed.** The route goes through the same
  `WrapService(... UserAccessValidator())` pipeline as the JSON favorite
  routes, so anonymous callers get the identical `401
  E-1-CMD-AUT-001` before the service runs.
- **Response transport: wrapped endpoint.** Nothing here needs a
  non-JSON transport, so the summary is a normal `WrapService` endpoint
  (not a raw handler like FEAT-003's CSV route) and is therefore visible
  to nexcommon's swagger generator.
- **No new component boundary.** The feature adds one DAO method, one
  service method, one DTO, one route registration, one adapter call and
  one tooltip. It does not introduce a new package, a new service, or a
  new datastore.
- **Route ordering.** `GET /v1/favorites/summary` is a static path
  distinct from `GET /v1/favorites`; the only variable path,
  `/v1/favorites/{item_id}`, is registered for `DELETE` only, so a `GET`
  cannot fall into it. If FEAT-004's `GET /v1/favorites/count` is present,
  the two static paths coexist without ordering concerns.

## Components

Backend, all in `sandbox-svc-code`:

- **`dao/favorite_dao.go`** (changed) — add `Summary(userID int64)
  (FavoriteSummary, error)` to the `FavoriteDAO` interface and its
  `favoriteDAO` implementation. One statement:

  ```sql
  SELECT COUNT(*) AS total,
         COUNT(*) FILTER (WHERE NOT COALESCE(c.is_available, FALSE)) AS discontinued
    FROM favorite_items f
    LEFT JOIN catalog_items c ON c.id = f.item_id
   WHERE f.user_id = $1
  ```

  `LEFT JOIN` and `COALESCE(..., FALSE)` mirror `dao.List`: a favorite
  whose catalog row is missing is still counted in `total` and treated
  as unavailable in `discontinued`, never dropped. `QueryRowContext(
  context.Background(), ...)` and `log.Error().Err(...)` on failure match
  the file's existing style.
- **`dao/type.go`** (changed) — add
  `FavoriteSummary struct { Total int64; Discontinued int64 }`. The DAO
  row type carries no JSON contract, same as the existing types.
- **`services/favorite/summary_favorite.go`** (new) — `Summary(ctx, _,
  _)` reads `ctx.AuthAccessTokenModel.AuthenticationServerUserID`, calls
  `s.dao.Summary(userID)`, and returns
  `(nil, svcout.NewFavoriteSummary(row.Total, row.Discontinued), nil)`.
  It reads no request input (REQ-010, REQ-011). It returns the raw DAO
  error for the formator to render as the standard `500` (REQ-017).
- **`dto/out/favorite.go`** (changed) — add
  `FavoriteSummaryResponse{ Total int64 \`json:"total"\`; Discontinued
  int64 \`json:"discontinued"\` }` and
  `NewFavoriteSummary(total, discontinued int64)
  FavoriteSummaryResponse`. Both are `int64`, so they encode as JSON
  integers, never strings or floats.
- **`endpoint/favorite_endpoint/favorite_endpoint.go`** (changed) — add
  `registerSummary()` and call it from `RegisterEndpoint()`:

  ```
  Path + "/summary", GET
  WrapService(NewWarpServiceParam(srv, srv.Summary,
      httpValidator.UserAccessValidator()).
    Tags(constanta.TagFavorite).
    OperationID("getFavoritesSummary").
    Description("Summarize the calling user's favorites: total rows and how many reference a discontinued catalog item.").
    DTOOut(http_validator.NewDTOParam(svcout.FavoriteSummaryResponse{})))
  ```

  This mirrors `registerList` exactly: plain `WrapService`, no body, no
  path params. The path constant is composed from the existing `Path`,
  so the final route is `/v1/favorites/summary`.
- **`api/openapi.yaml`** and `postman/` (changed) — the wrapped route
  appears in nexcommon's generated `/inspect` docs, so `/go-dev`
  regenerates or hand-adds the path, the request/response schemas and the
  standard envelope there, following the repo's established flow.
  `/go-release` diffs this file for drift and breaking changes.
- **No migration file.** No new table, column, index or constraint.

Untouched: the three existing favorite routes and their service methods,
FEAT-003's CSV route, the auth/token stack, the metrics middleware, the
schema, and the DAO methods that already exist.

## Authorization

- **Who may call it:** any authenticated caller. There is no role,
  permission or menu gate beyond the shared
  `httpValidator.UserAccessValidator()`; anonymous or invalid tokens get
  `401 E-1-CMD-AUT-001` before the service runs (REQ-013, REQ-022).
- **Row scope:** the user id is read only from
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`. The endpoint
  declares no query, path or body parameters, and the service reads
  nothing else, so no request input can widen or redirect the count
  (REQ-010, REQ-011, REQ-020, REQ-026).
- **No administrative or cross-user view exists** — the spec's Non-Goals.
- **Filter-options principle:** not applicable. This feature adds no
  list page and no filter; `GET /v1/favorites` is unchanged and has no
  filter parameters. There is therefore no filter-options endpoint to
  authorize separately.

## Backend — Endpoints

- `GET /v1/favorites/summary` — `consumer: fe`. Returns the calling
  user's own summary as `payload.total` and `payload.discontinued` in
  the standard success envelope. No request parameters and no body;
  anything sent is unbound and ignored. Authenticated by the shared
  `UserAccessValidator()`.

This is the whole API surface this feature adds; no existing endpoint
changes. The `fe` consumer is the `AppShell` badge tooltip described in
`## Frontend`, which covers every signed-in screen (there is no separate
page for it).

## API contract (consumer-facing)

| Method | Path | Consumer | Success | Errors |
|---|---|---|---|---|
| `GET` | `/v1/favorites/summary` | `fe` | `200`, `Content-Type: application/json`, body `{"success": true, "header": {...}, "payload": {"total": <integer>, "discontinued": <integer>}}` | `401` standard envelope (anonymous/invalid token); `500` standard envelope (`E-5-SBX-SRV-001`) on unexpected failure. No `404` — zero favorites is a valid `200` with `{0, 0}` (REQ-012). |

Success envelope, exactly (the shape nexcommon's writer emits):

```json
{"success": true,
 "header": {"request_id": "...", "version": "...", "timestamp": "..."},
 "payload": {"total": 3, "discontinued": 1}}
```

Error envelope, exactly:

```json
{"success": false,
 "header": {"request_id": "...", "version": "...", "timestamp": "..."},
 "payload": {"status": 401, "code": "E-1-CMD-AUT-001", "message": "..."}}
```

`total` and `discontinued` are JSON integers (`int64`), never strings or
floats, non-negative by construction, with `0 <= discontinued <= total`.
The FE reads `payload.total` and `payload.discontinued`. See
"Documented deviations" items 1–2 for why the spec's literal bodies
differ from this.

## Data design

**No schema change** (REQ-019). The feature reads FEAT-001's
`favorite_items` and FEAT-003's `catalog_items` columns and adds no
table, column, index, constraint or migration; see `database.md`, which
records the schema-designer position explicitly (all three phases are
"not applicable", and no batch is produced).

The aggregate query joins `favorite_items` to `catalog_items` on the
latter's primary key, filtered by `favorite_items.user_id` — the leading
column of the existing composite PK `(user_id, item_id)`. No new index
is justified: the per-user row set is capped at 500 by FEAT-001's cap,
and the join is a PK lookup per row. The query is a single read-only
statement, so it is safe to run concurrently with add/remove; a
concurrent mutation can only change which correct value the caller sees.

`ERD.md` is unchanged: no table is added or altered by this feature, so
no row is added to its `## Tables by feature` table.

## Spec traceability

- **REQ-001** — `GET /v1/favorites/summary` route in
  `favorite_endpoint.registerSummary` → `favorite.Service.Summary` →
  `favoriteDAO.Summary` → one aggregate `SELECT` over the caller's rows.
  Covered.
- **REQ-002** — `200` + `Content-Type: application/json` with
  `payload.total` and `payload.discontinued` as JSON integers; zero
  favorites is a normal `200` with `{0, 0}`. **Deviation:** the body is
  nexcommon's `{success, header, payload}` envelope, not the literal
  `{"total": N, "discontinued": M}` — item 1 below. Covered as the
  envelope.
- **REQ-003** — `COUNT(*)` over `favorite_items` has no availability
  predicate, so discontinued favorites are included in `total`. Covered.
- **REQ-004** — `discontinued` is `COUNT(*) FILTER (WHERE NOT
  COALESCE(c.is_available, FALSE))`, i.e. exactly the rows whose catalog
  item is currently soft-marked unavailable, with a missing catalog row
  degrading to unavailable as the list already does. Covered.
- **REQ-005** — `formatFavoriteSummary` renders the literal
  `N favorites (M discontinued)` from the exact `int64` values; no
  `99+` cap is applied. Covered.
- **REQ-006** — The Ant Design `Tooltip` wraps the badge span, which
  FEAT-004 renders only when its count is positive and known. A hidden
  badge has no trigger, so hovering its area shows no tooltip and starts
  no fetch. Covered.
- **REQ-007** — `ensureSummary()` is called only from the tooltip's
  `onOpenChange(true)` (hover), never on mount; `FavoriteCountProvider`
  fetches the summary nowhere else. A fetched summary is refreshed only
  via `refreshSummary()`, which is called from the two success paths.
  Repeat hover is a no-op while `'ready'`. Covered.
- **REQ-008** — `open={hovered && summary !== null}` means no tooltip
  while idle/loading/error; the badge itself is untouched. A later hover
  retries when the state is `'error'` and shows the tooltip once a fetch
  succeeds. Covered.
- **REQ-009** — Both values are computed per request by the aggregate
  query; there is no stored counter, cache or denormalized column, and
  the Non-Goals forbid adding one. Covered.
- **REQ-010** — The only `WHERE` clause is `f.user_id = <context user
  id>`; the service reads the id only from
  `ctx.AuthAccessTokenModel.AuthenticationServerUserID`. Covered.
- **REQ-011** — The endpoint declares no query, path or body parameters;
  the DAO takes only the context user id. No request input exists to
  bind or override. Covered.
- **REQ-012** — `COUNT(*)` returns `0` for a user with no rows; there is
  no not-found branch, so the response is `200` with `{0, 0}`. Covered.
- **REQ-013** — `UserAccessValidator()` returns `errors.ErrUnauthorized`
  before the service runs; the formatter emits `401` with code
  `E-1-CMD-AUT-001` and no summary payload. **Deviation:** the envelope
  is `{success, header, payload{status, code, message}}`, not
  `{"error_code", "message"}` — item 2 below. Covered as the envelope.
- **REQ-014** — The same aggregate query can return `total ==
  discontinued` when every referenced item is unavailable; no filter
  removes those rows from `total`. Covered.
- **REQ-015** — The tooltip only wraps FEAT-004's badge; when the badge
  is hidden there is no trigger and `ensureSummary()` cannot run. Covered.
- **REQ-016** — Controlled `open` plus no spinner/error UI anywhere: the
  provider stores only `summary` and a status string, and re-renders
  nothing on the badge. Covered.
- **REQ-017** — Any unexpected DAO/service error is returned raw and
  rendered by nexcommon's formatter as `500 E-5-SBX-SRV-001` with no
  partial summary; the FE maps it to `'error'` and shows no tooltip.
  Covered.
- **REQ-018** — `refreshSummary()` synchronously clears the held values
  before re-fetching, and it is called from both success paths; a failed
  refresh leaves `'error'` with no values, so pre-action values are
  unreachable. Covered.
- **REQ-019** — `database.md` records no schema change; the query reads
  the existing `favorite_items` and `catalog_items` columns only.
  Covered.
- **REQ-020** — Same implementation as REQ-010: ownership is satisfied
  by the only `WHERE` clause, with no request-supplied user id and no
  per-row `404` because no individual row is addressed. Covered.
- **REQ-021** — The tooltip attaches to FEAT-004's badge and reuses its
  provider, visibility rule and refresh trigger; the badge's count
  source, `99+` format and hide-at-zero behaviour are unchanged. Covered.
- **REQ-022** — The route reuses the same `UserAccessValidator()` and
  Redis token store as the existing favorite routes; no login/session
  code is added. Covered.
- **REQ-023** — `refreshSummary()` is called from the same two success
  branches that FEAT-004's `refresh()` uses (`useFavoriteStar.toggle`
  and `useFavorites.remove`), only when a summary had been fetched.
  Covered.
- **REQ-024** — For any `0 <= D <= T` the DAO returns exactly `T` and
  `D` and the DTO carries them unmodified; `T = 0` gives `{0, 0}`.
  Covered.
- **REQ-025** — Same code path as REQ-013: `401` with the standard
  envelope and no summary payload. Covered.
- **REQ-026** — No request input is read for identity: the endpoint binds
  no parameters and the service reads only the auth context. Covered.
- **REQ-027** — The badge keeps FEAT-004's `99+` display; the tooltip
  formatter always uses the exact `total` and `discontinued` integers.
  Covered.
- **REQ-028** — Loading, failed fetch and `500` all map to a null
  summary / non-ready status, so `open` is false; no spinner, error icon
  or toast exists and the badge is not re-rendered by summary state.
  Covered.
- **REQ-029** — After either success path, the summary is invalidated
  and re-fetched; the next tooltip therefore shows the new values, and
  the pre-action values are never rendered. Covered.

Non-goals, all recorded as not applicable: no click-through, no
polling/realtime/cross-tab sync, no server-side or persisted cache, no
per-item breakdown, no admin/cross-user summary, no change to FEAT-004's
badge, no new page/settings/notifications, no tooltip trigger other than
hover, no localization or pluralization, no animation, no analytics, and
no new table/column/index/cache.

## Planned breaking changes

None. This feature adds one route and one response payload; no existing
endpoint, response field, or table definition is removed, renamed or has
its required-ness changed.

## Implementation Plan

- PART-1: Summary read path — `dao.FavoriteDAO.Summary` (interface + implementation), `dao.FavoriteSummary`, `services/favorite/summary_favorite.go`, `dto/out.FavoriteSummaryResponse` with `NewFavoriteSummary`, plus unit tests for zero/zero, one/zero, one/one, mixed and all-discontinued rows, the 500-row cap, two-user isolation, a missing catalog row, and the auth-context user id being the only scope.
- PART-2: HTTP route and contract — `favorite_endpoint.registerSummary` on `GET <Path>/summary`, its call from `RegisterEndpoint()`, the OpenAPI/Postman artifact for the route, and handler tests for the `200` payload, the `401` envelope, the `500` envelope and the `0` case; plus the `/metrics` scrape check in `## Observability`.

PART-2 depends on PART-1. There is **no frontend PART ID**: `/go-release`
maps every PART ID here to a merged `/go-dev` PR in `sandbox-svc-code`,
and `/fe-dev` records no PART in `dev-test-results.md`, so a
frontend-only PART would block the backend release with no PR able to
satisfy it. The tooltip work in `sandbox-svc-ui` is sequenced by
`/fe-dev` from `## Frontend` above, after FEAT-004's badge exists and
PART-2's contract exists.

## Observability

- **Current state** — the surveyed branch already has the full metrics
  stack: `/metrics` exposed in `router.InitHttpService` outside token
  validation, `middleware.StatusAwareMetricsMiddleware` installed via
  `router.Use(metricMiddleware.Serve)` reading the real captured status,
  and nexcommon `v0.3.68` whose writer no longer hardcodes 200.
  FEAT-003's live run confirms status-labelled samples (`200`/`401`) for
  its route. Nothing needs wiring.
- **Decision** — **reuse** the existing instrumentation; add no
  collector, registry or dashboard. The new route is registered on the
  same `*mux.Router`, so it is observed with no extra code. No
  feature-specific metric: the spec states there is no post-launch
  business metric for this feature, and the standard request metric
  already distinguishes a `200` summary from a `401`.
- **Implementation boundary** — inbound HTTP only, on
  `GET /v1/favorites/summary`. The label set stays bounded: the
  middleware resolves the template via
  `mux.CurrentRoute(r).GetPathTemplate()`, which is the literal
  `/v1/favorites/summary` — no ids, no user ids, no query strings. The
  `total`/`discontinued` values are never labels. `/metrics` keeps its
  existing deployment-level access restriction.
- **Verification** — during implementation, scrape `/metrics` after an
  authenticated summary call and confirm a `200` sample for the route
  template, then after an anonymous call confirm a `401` sample for the
  same route (not folded into `200`), and confirm the existing
  `/v1/favorites` samples are not duplicated or relabelled by the new
  registration.

## Documented deviations from the spec's wording

Recorded here so a later docs pass can correct the source wording.
**No blocking `code_prep_feedback` entry is filed for these** — the user
asked that open points be picked with the recommended option rather than
blocking the design, and none of these changes what is being built. They
follow the same deviations FEAT-003 and FEAT-004 already recorded.

1. **Success body shape.** spec.md REQ-002/REQ-024 and testcase.md
   TC-001–TC-008 assert a literal body `{"total": N, "discontinued": M}`.
   nexcommon's wrapped endpoint always emits
   `{"success": true, "header": {...}, "payload": {"total": N,
   "discontinued": M}}`; the two values are unchanged and the status is
   `200`. Recommended fix: a later `/testcase-generation` /
   `/document-preparation` pass should assert `200` +
   `payload.total`/`payload.discontinued`, or state that the body is the
   standard success envelope.
2. **401 body shape and code.** spec.md REQ-013 and TC-009/TC-010
   describe `{"error_code": "...", "message": "..."}`. The real response
   is `{"success": false, "header": {...}, "payload": {"status": 401,
   "code": "E-1-CMD-AUT-001", "message": "..."}}`. This is the same
   correction FEAT-003 and FEAT-004 already recorded. Recommended fix:
   assert the `401` status and the standard envelope, optionally naming
   `E-1-CMD-AUT-001`.
3. **FEAT-004's frontend badge is not implemented yet.** FEAT-005 adds
   hover behaviour to FEAT-004's badge and reuses its
   `FavoriteCountProvider`. The FEAT-004 design is merged, but the
   surveyed FE tree has no provider or badge. Sequence FEAT-004's
   frontend delivery before FEAT-005's, or have `/fe-dev` record a
   `dev_feedback` entry rather than build a duplicate badge/provider.
4. **Backend code baseline is not `release_branch`.** `release_branch`
   is the user-confirmed `main`, which still holds only `README.md`.
   The real code lives on `feat/FEAT-003-dev-export-favorites-csv`, whose
   PR 2 is open against `feat/FEAT-003` and not merged. `/go-dev` must
   start from a commit that contains FEAT-001's implementation plus the
   FEAT-003 code (the user's instruction was to survey the newest branch
   with the real code), and `/go-release` should not expect a releasable
   `main` until that chain merges. This is a repository-state
   precondition, not a design change; the design itself depends only on
   FEAT-001's schema and DAO/service/endpoint patterns.
5. **"Header badge" is the shell brand row.** The UI has no `<header>`
   element; FEAT-004 places the badge in `AppShell`'s brand row. FEAT-005
   attaches the tooltip there. REQ-005/REQ-006's intent is satisfied by
   the badge's actual location.
6. **FEAT-001's frontend calls a contract the backend does not serve.**
   `favoriteAdapter.js` calls `/favorites` (no `/v1`) and
   `favoriteService.js` reads a paged `data/page/page_size/total`
   envelope; the real backend serves `/v1/favorites` with
   `payload.favorites`. FEAT-005's summary adapter uses `/v1` and
   `payload.total`/`payload.discontinued` and does not copy the mistake.
   Fixing FEAT-001's three adapters is a separate reconciliation;
   `/fe-dev` should record it as a `dev_feedback` entry rather than
   silently work around it.
7. **No `UIMap.md`.** The spec has `has_ui: true` but
   `/document-preparation` produced only `PRD.md` and `AppFlow.md`. This
   design therefore names routes, components, states and behaviour but
   not visual specifics (tooltip placement, delay, styling). The
   requirements above are sufficient for `/fe-dev` either way.
