---
feature_code: FEAT-001
service: sandbox-svc
spec_version: 3
last_updated: 2026-10-10
---

# FEAT-001 — Favorite Items (PRD)

## Overview

Users browsing the sandbox-svc catalog have no way to quickly get back to
items they're interested in — they have to re-search every time. This
feature adds a personal "favorites" list so a user can mark items of
interest and return to them directly. See
[spec.md](spec.md) for the full requirement detail.

## Product Goal

Let a logged-in user save items of interest and come back to exactly
that set later, without re-searching, while keeping the feature simple
for this first pass (no sharing, no recommendations, no sort/filter yet).

## Features

| Feature | Description | Expected behavior |
|---|---|---|
| Favorite an item | Star icon on the item detail page | Item is added to the user's personal favorites list immediately (REQ-001) |
| Un-favorite an item | Available from the item detail page or the favorites list | Item is removed from the list immediately, no reload (REQ-003) |
| My Favorites page | Lists everything the user has favorited | Shows all favorited items, including discontinued ones marked "unavailable" — catalog items are never hard-deleted, only soft-marked, so this always holds (REQ-002, REQ-004) |
| Idempotent favoriting | Favoriting an already-favorited item | No duplicate record, no error (REQ-005) |
| Anonymous gating | Anonymous user attempts to favorite | Redirected to login, cannot favorite (REQ-006) |
| Favorites cap | A user tries to exceed 500 favorites | Clear error on the 501st attempt; no silent failure or eviction (REQ-007) |
| Row ownership enforcement | A user attempts to view/un-favorite/edit another user's favorite row | Request is rejected with a uniform 404 whether the row belongs to someone else or doesn't exist at all — never a 403, so existence can't be inferred (REQ-008) |

Deferred to a later pass: sorting/filtering the favorites list.
