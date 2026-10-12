package dao

import "time"

// FavoriteItem is one row of the "My Favorites" list: the (user, item) relation
// joined to the item's *current* catalog state.
//
// Name and Available are read live from catalog_items at list-time, never
// denormalized onto favorite_items at favorite-time (REQ-004) — an item that
// is discontinued after it was favorited still lists, with Available false.
type FavoriteItem struct {
	ItemID      int64
	Name        string
	Available   bool
	FavoritedAt time.Time
}

// AddOutcome is what favoriteDAO.Add did. The three cases are genuinely
// distinct to the caller: two are success responses (REQ-005 makes a repeat
// favorite idempotent, not an error) and one is REQ-007's cap failure.
type AddOutcome int

const (
	// AddInserted — the item was not favorited and now is.
	AddInserted AddOutcome = iota
	// AddAlreadyFavorited — the item was already favorited; nothing changed and
	// nothing errored (REQ-005).
	AddAlreadyFavorited
	// AddLimitReached — the user is already at the cap, the item was not already
	// favorited, and nothing was inserted or evicted (REQ-007).
	AddLimitReached
)
