---
feature_code: FEAT-001
service: sandbox-svc
spec_version: 3
last_updated: 2026-10-10
---

# FEAT-001 — App Flow

<!-- BEGIN:DOCPREP -->
## Overview

A logged-in user can favorite/un-favorite catalog items and review their
saved items on a dedicated "My Favorites" page. Anonymous users are
redirected to login rather than allowed to favorite.

## Key User Journeys

- **Favorite an item (REQ-001):** User is viewing an item detail page →
  taps the "favorite" star icon → item appears on their My Favorites
  page immediately, no reload.
- **Un-favorite an item (REQ-003):** User is on the item detail page or
  the My Favorites page → taps to un-favorite → item disappears from
  My Favorites immediately, no reload.
- **Browse favorites, including a discontinued one (REQ-002, REQ-004):**
  User opens My Favorites → sees every item they've favorited; a
  discontinued item still appears, marked "unavailable," rather than
  silently vanishing or erroring — catalog items are never hard-deleted,
  only soft-marked, so this case always resolves cleanly.
- **Anonymous user tries to favorite (REQ-006):** Anonymous visitor taps
  the favorite star → redirected to login instead of favoriting.
- **Hit the favorites cap (REQ-007):** User already has 500 favorites →
  attempts to favorite a 501st item → sees a clear error; nothing is
  silently evicted or dropped.
- **Row ownership enforcement (REQ-008):** Invisible to a legitimate user
  under normal use — every view/un-favorite (and, via FEAT-002, note
  edit/clear) request is checked against the authenticated user id
  before it's allowed. A request against another user's row, or a
  nonexistent one, gets the same uniform 404 either way.

## Main Pages / Actions

- Item detail page — favorite/un-favorite star icon.
- My Favorites page — list of all favorited items, with per-item
  un-favorite action and an "unavailable" marker for discontinued items.
<!-- END:DOCPREP -->
