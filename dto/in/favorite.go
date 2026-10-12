package in

// AddFavoriteRequest is the body of POST /favorites.
//
// The user is deliberately not a field: it comes from the access token, so a
// caller has no way to favorite on someone else's behalf (REQ-008).
//
// required:"favorite.create" gates the block on the route's menu, which is what
// makes item_id mandatory here — with min:"1" alongside it, a missing/zero
// item_id answers 400 E-4-CMD-DTO-001 ("ITEM_ID Cannot Empty") and a negative
// one answers 400 E-4-CMD-DTO-003, rather than falling through to a confusing
// not-found.
type AddFavoriteRequest struct {
	ItemID int64 `json:"item_id" required:"favorite.create" min:"1"`
}
