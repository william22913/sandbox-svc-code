package out

import "time"

// FavoriteItem is one entry of GET /favorites' payload.
//
// Name and Available are the item's state read at list-time, not a copy taken
// when it was favorited (REQ-004): an item discontinued after favoriting still
// appears, with Available false.
//
// FavoritedAt is a plain time.Time, deliberately not nexcommon's
// out.DefaultTime: DefaultTime's MarshalJSON returns (nil, nil) for a zero
// value, which makes json.Marshal fail the whole response with "unexpected end
// of JSON input" — the controller logs that and then writes a 200 with an empty
// body. Plain time.Time cannot fail that way.
type FavoriteItem struct {
	ItemID      int64     `json:"item_id"`
	Name        string    `json:"name"`
	Available   bool      `json:"available"`
	FavoritedAt time.Time `json:"favorited_at"`
}

// AddFavoriteResponse is POST /favorites' payload.
//
// AlreadyFavorited is what makes REQ-005's idempotency observable to a client:
// favoriting an item that was already in the list is a success with this true,
// not an error and not a second row.
type AddFavoriteResponse struct {
	ItemID           int64 `json:"item_id"`
	AlreadyFavorited bool  `json:"already_favorited"`
}

// RemoveFavoriteResponse is DELETE /favorites/{item_id}'s payload, echoed back
// so the caller can confirm which item left the list.
type RemoveFavoriteResponse struct {
	ItemID int64 `json:"item_id"`
}

// FavoriteSummaryResponse is GET /favorites/summary's payload: the caller's
// favorite count and how many of those items are currently unavailable.
//
// Both are integers, never the FEAT-004 badge's "99+" string — the cap is a
// display rule for the badge and does not apply to the summary (REQ-005).
type FavoriteSummaryResponse struct {
	Total        int64 `json:"total"`
	Discontinued int64 `json:"discontinued"`
}

// NewFavoriteSummary builds the payload. No clamping: the DAO's single
// statement guarantees 0 <= discontinued <= total.
func NewFavoriteSummary(total, discontinued int64) FavoriteSummaryResponse {
	return FavoriteSummaryResponse{Total: total, Discontinued: discontinued}
}

// FavoriteList is GET /favorites' payload. A wrapper rather than a bare array
// so FEAT-002's per-favorite note (and the deferred sort/filter pass) can add a
// sibling field or a page envelope without changing this response's shape.
type FavoriteList struct {
	Favorites []FavoriteItem `json:"favorites"`
}

// NewFavoriteList builds the payload, guaranteeing an empty list serialises as
// [] rather than null (TC-004: an empty "My Favorites" page is a normal state,
// not a missing one).
func NewFavoriteList(items []FavoriteItem) FavoriteList {
	if items == nil {
		items = make([]FavoriteItem, 0)
	}

	return FavoriteList{Favorites: items}
}
