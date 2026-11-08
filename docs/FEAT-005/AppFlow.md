---
feature_code: FEAT-005
service: sandbox-svc
spec_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — App Flow

<!-- BEGIN:DOCPREP -->
## Overview

A signed-in user with the FEAT-004 header badge visible can hover it to see
`N favorites (M discontinued)`, served by `GET /v1/favorites/summary`. The
summary is fetched on first hover and refreshed after a favorite or
un-favorite; the tooltip is hidden while loading, on error, and whenever the
badge itself is hidden.

## Key User Journeys

- **See the favorites breakdown (REQ-001, REQ-002, REQ-003, REQ-004, REQ-005):** A signed-in user with at least one favorite hovers the visible header badge → the tooltip shows `N favorites (M discontinued)` with that user's own exact totals.
- **No discontinued favorites (REQ-004, REQ-005):** A user whose favorites are all still available hovers the badge → the tooltip reads `N favorites (0 discontinued)`.
- **No favorites yet (REQ-012, REQ-015):** A user with zero favorites has no badge (FEAT-004) → there is nothing to hover and no tooltip.
- **Favorite or un-favorite (REQ-007, REQ-018):** The user favorites or un-favorites an item → an already-fetched summary is refreshed, and the next tooltip shows the new counts, never the pre-action ones.
- **Summary unavailable (REQ-008, REQ-016, REQ-017):** While the summary is loading or after a fetch error → hovering shows no tooltip; the badge's own count display is unchanged.
- **Anonymous caller (REQ-013):** A caller without a valid session hits the endpoint → receives the standard `401` error envelope and no summary body.

## Main Pages / Actions

- The shared app header on every signed-in screen that shows the FEAT-004 badge (no new page).
- Action: hover the visible badge to fetch and display `GET /v1/favorites/summary`.
- Action: after a successful favorite/un-favorite, refresh an already-fetched summary.
- The tooltip is display-only; it has no click action.
<!-- END:DOCPREP -->

<!-- BEGIN:CODEPREP -->
## Routes and Components

- **No new frontend route.** The tooltip attaches to FEAT-004's badge in
  `AppShell`, so it is available on the existing signed-in routes
  `/favorites`, `/catalog`, `/catalog/:itemId` and the index redirect.
  `/login` is outside the shell, so an anonymous visitor never renders
  the badge or the tooltip.
- **Backend endpoint:** `GET /v1/favorites/summary` (`consumer: fe`),
  authenticated by the shared `UserAccessValidator()`, accepting no
  request input, returning `payload.total` and `payload.discontinued` in
  the standard `{success, header, payload}` envelope.
- **Components changed:** `FavoriteCountProvider` (summary state,
  first-hover fetch, post-action invalidation), `AppShell` (Ant Design
  `Tooltip` around the existing badge), `favoriteAdapter` /
  `favoriteService` (summary call and response mapper), and
  `useFavoriteStar` / `useFavorites` (refresh the already-fetched summary
  after a successful action).
- **Tooltip visibility:** shown only while the badge is visible and the
  summary is ready. Hidden while the badge is hidden, while the summary
  is idle, loading or failed, and after a favorite/un-favorite until the
  refresh completes. Text is the literal `N favorites (M discontinued)`
  with exact integers; the badge's own `99+` cap never applies to `N`.
- **Unaffected:** FEAT-004's badge count, `99+` cap, visibility and
  loading/error behaviour; the existing favorite routes and their
  payloads; the schema (no migration).
<!-- END:CODEPREP -->
