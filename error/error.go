package error

import (
	stderrors "errors"

	error2 "github.com/nexsoft-git/nexcommon/error"
)

// Business error codes. Declared separately so the bundle guard test can
// reference the same string the sentinel is built from.
const (
	CodeFavoritesLimitReached = "FAVORITES_LIMIT_REACHED"
	CodeFavoriteNotFound      = "FAVORITE_NOT_FOUND"
	CodeCatalogItemNotFound   = "CATALOG_ITEM_NOT_FOUND"
)

// One typed sentinel per business failure. Returning one of these is the only
// way the real status/code/message reaches the client — any other error type is
// coerced to a generic 500 by nexcommon's Formator.
//
// These carry no parameters, so they are safe to return as package-level vars:
// the shared-sentinel data race only bites callers that use .Param(...), which
// mutates the sentinel in place.
var (
	// ErrFavoritesLimitReached is REQ-007's clear error: the caller already has
	// 500 favorites. Nothing is inserted and nothing is evicted.
	ErrFavoritesLimitReached = error2.NewUnBundledErrorMessages(
		422,
		stderrors.New(CodeFavoritesLimitReached),
		nil,
	)

	// ErrFavoriteNotFound is REQ-008's uniform 404 — returned identically when
	// the row belongs to another user and when it does not exist at all, so the
	// two cases are indistinguishable to the caller.
	ErrFavoriteNotFound = error2.NewUnBundledErrorMessages(
		404,
		stderrors.New(CodeFavoriteNotFound),
		nil,
	)

	// ErrCatalogItemNotFound is a POST /favorites naming an item the catalog
	// does not have. architecture.md's traceability table assigns this check to
	// favorite_service.Add ("Read-only join in favorite_service.List/.Add (to
	// validate item exists)"), so it is a real 404 rather than the foreign-key
	// violation that would otherwise surface — and be flattened into a generic
	// 500 — when the insert runs.
	ErrCatalogItemNotFound = error2.NewUnBundledErrorMessages(
		404,
		stderrors.New(CodeCatalogItemNotFound),
		nil,
	)
)

// BundledCodes is the codes this service's common.error bundle must carry that
// cannot be derived from nexcommon's source: its own business codes, plus the
// formator's configured default for anything unexpected.
//
// The codes nexcommon itself emits (its 22 NewUnBundledErrorMessages sentinels,
// including the auth family the token validators return) are NOT listed here —
// they are read out of the vendored message.go by the bundle guard test, so
// that list cannot drift from the library the way a hand-maintained one did.
var BundledCodes = []string{
	CodeFavoritesLimitReached,
	CodeFavoriteNotFound,
	CodeCatalogItemNotFound,

	// Set via .DefaultInternalCode(...) in server_config — nexcommon's own
	// default is E-5-CMD-SRV-001, which this service overrides.
	"E-5-SBX-SRV-001",
}
