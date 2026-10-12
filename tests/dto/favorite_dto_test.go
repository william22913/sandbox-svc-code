// Package dto holds the request/response DTO contract tests. No config import:
// config's init() calls log.Fatal on a bad environment, which would kill the
// test binary at import time.
package dto

import (
	"encoding/json"
	"strings"
	"testing"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"

	"github.com/william22913/sandbox-svc-code/dto/in"
	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	svcerror "github.com/william22913/sandbox-svc-code/error"
)

const menu = "favorite.create"

func validate(t *testing.T, req *in.AddFavoriteRequest, menu string) error {
	t.Helper()
	return tag_validator.NewTagValidator().ValidateByTag(internalCtx.NewContextModel(), req, "", menu)
}

// TestAddRequestRequiresItemID — a missing/zero item_id is a 400 DTO-empty
// error, not a silent zero that would reach the DAO.
func TestAddRequestRequiresItemID(t *testing.T) {
	err := validate(t, &in.AddFavoriteRequest{}, menu)
	if err == nil {
		t.Fatal("empty item_id validated, want the DTO-empty error")
	}
	if !strings.Contains(err.Error(), "E-4-CMD-DTO-001") {
		t.Fatalf("empty item_id error = %q, want E-4-CMD-DTO-001", err.Error())
	}
}

// TestAddRequestRejectsNonPositiveItemID — min:"1" catches a negative id before
// it reaches the database, where it would only be a confusing not-found.
func TestAddRequestRejectsNonPositiveItemID(t *testing.T) {
	err := validate(t, &in.AddFavoriteRequest{ItemID: -5}, menu)
	if err == nil {
		t.Fatal("negative item_id validated, want the DTO-rule error")
	}
	if !strings.Contains(err.Error(), "E-4-CMD-DTO-003") {
		t.Fatalf("negative item_id error = %q, want E-4-CMD-DTO-003", err.Error())
	}
}

func TestAddRequestAcceptsPositiveItemID(t *testing.T) {
	if err := validate(t, &in.AddFavoriteRequest{ItemID: 7}, menu); err != nil {
		t.Fatalf("valid item_id rejected: %v", err)
	}
}

// TestAddRequestValidationIsMenuGated — required:"favorite.create" is what makes
// item_id mandatory, so a route carrying a different menu must not enforce it.
// This guards the tag/DTO pairing the endpoint layer depends on.
func TestAddRequestValidationIsMenuGated(t *testing.T) {
	if err := validate(t, &in.AddFavoriteRequest{}, "some.other.menu"); err != nil {
		t.Fatalf("validation ran outside its menu: %v", err)
	}
}

// TestFavoriteListEmptyIsArrayNotNull — TC-004's empty state must serialise as
// [] so a client can tell "no favorites" from a broken response.
func TestFavoriteListEmptyIsArrayNotNull(t *testing.T) {
	for _, name := range []string{"nil", "empty"} {
		items := []svcout.FavoriteItem(nil)
		if name == "empty" {
			items = []svcout.FavoriteItem{}
		}

		raw, err := json.Marshal(svcout.NewFavoriteList(items))
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if got := string(raw); got != `{"favorites":[]}` {
			t.Fatalf("%s: list = %s, want {\"favorites\":[]}", name, got)
		}
	}
}

// TestFavoriteItemMarshalsWithZeroTime — regression guard for the reason this
// DTO does not use nexcommon's out.DefaultTime: that type fails json.Marshal
// outright on a zero value, which the controller turns into a 200 with an empty
// body.
func TestFavoriteItemMarshalsWithZeroTime(t *testing.T) {
	raw, err := json.Marshal(svcout.FavoriteItem{ItemID: 10, Name: "item-10", Available: true})
	if err != nil {
		t.Fatalf("marshal with zero FavoritedAt: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	for _, key := range []string{"item_id", "name", "available", "favorited_at"} {
		if _, ok := got[key]; !ok {
			t.Errorf("response is missing %q: %s", key, raw)
		}
	}
}

// TestErrorSentinelsCarryTheirOwnCode — the DTO layer and the service layer must
// answer with the codes the bundles carry. With no Formator built (as here) the
// sentinel renders its raw code, which is exactly the value the bundle guard
// then requires an entry for.
func TestErrorSentinelsCarryTheirOwnCode(t *testing.T) {
	cases := map[string]string{
		svcerror.CodeFavoritesLimitReached: svcerror.ErrFavoritesLimitReached.Error(),
		svcerror.CodeFavoriteNotFound:      svcerror.ErrFavoriteNotFound.Error(),
		svcerror.CodeCatalogItemNotFound:   svcerror.ErrCatalogItemNotFound.Error(),
	}

	for want, got := range cases {
		if got != want {
			t.Errorf("sentinel renders %q, want %q", got, want)
		}
	}
}
