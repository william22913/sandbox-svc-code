// Package service tests the favorite service layer's outcome -> response
// mapping against a fake DAO.
//
// The real SQL is covered by tests/db against a live Postgres; what is worth
// asserting here is everything above the statements: which user id the service
// reads out of the request context, which typed sentinel each DAO outcome turns
// into, and that an unexpected DAO failure is passed through rather than being
// dressed up as a business error the client would act on wrongly.
//
// No config import — config's init() calls log.Fatal on a bad environment,
// which would kill the test binary before any test runs.
package service

import (
	stderrors "errors"
	"testing"
	"time"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"

	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/dto/in"
	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	svcerror "github.com/william22913/sandbox-svc-code/error"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

const authUserID int64 = 42

// fakeDAO records what the service asked for and answers with what the test
// configured, so both the decision and the delegation are observable.
type fakeDAO struct {
	itemExists func(int64) (bool, error)
	add        func(userID, itemID, limit int64) (dao.AddOutcome, error)
	remove     func(userID, itemID int64) (bool, error)
	list       func(userID int64) ([]dao.FavoriteItem, error)

	addCalls    int
	addUserID   int64
	addItemID   int64
	addLimit    int64
	removeCalls int
	removeUser  int64
	removeItem  int64
	listCalls   int
	listUserID  int64
}

func (f *fakeDAO) Add(userID, itemID, limit int64) (dao.AddOutcome, error) {
	f.addCalls++
	f.addUserID, f.addItemID, f.addLimit = userID, itemID, limit
	if f.add == nil {
		return dao.AddInserted, nil
	}
	return f.add(userID, itemID, limit)
}

func (f *fakeDAO) Remove(userID, itemID int64) (bool, error) {
	f.removeCalls++
	f.removeUser, f.removeItem = userID, itemID
	if f.remove == nil {
		return false, nil
	}
	return f.remove(userID, itemID)
}

func (f *fakeDAO) List(userID int64) ([]dao.FavoriteItem, error) {
	f.listCalls++
	f.listUserID = userID
	if f.list == nil {
		return nil, nil
	}
	return f.list(userID)
}

func (f *fakeDAO) ItemExists(int64) (bool, error) { return true, nil }

// itemMissingDAO is the one behaviour Add needs that the recording fake above
// answers with "present" by default.
type itemMissingDAO struct{ fakeDAO }

func (f *itemMissingDAO) ItemExists(int64) (bool, error) { return false, nil }

func authContext() *internalCtx.ContextModel {
	ctx := internalCtx.NewContextModel()
	ctx.AuthAccessTokenModel.AuthenticationServerUserID = authUserID
	return ctx
}

func addRequest(itemID int64) *in.AddFavoriteRequest {
	return &in.AddFavoriteRequest{ItemID: itemID}
}

// TestAddInserted — REQ-001 / TC-001's backend half: a fresh favorite succeeds
// and is not reported as already present.
func TestAddInserted(t *testing.T) {
	fake := &fakeDAO{}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Add(authContext(), model.URLParam{}, addRequest(10))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	resp, ok := payload.(svcout.AddFavoriteResponse)
	if !ok {
		t.Fatalf("payload type = %T, want out.AddFavoriteResponse", payload)
	}
	if resp.ItemID != 10 || resp.AlreadyFavorited {
		t.Fatalf("payload = %+v, want item 10 with AlreadyFavorited false", resp)
	}
}

// TestAddDelegatesTheAuthenticatedUserAndTheCap — REQ-007 and REQ-008: the user
// comes from the token, never the request, and the cap handed to the DAO is the
// documented 500.
func TestAddDelegatesTheAuthenticatedUserAndTheCap(t *testing.T) {
	fake := &fakeDAO{}
	svc := favorite.NewService(fake)

	if _, _, err := svc.Add(authContext(), model.URLParam{}, addRequest(10)); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if fake.addUserID != authUserID {
		t.Errorf("DAO Add got user %d, want the token's user %d", fake.addUserID, authUserID)
	}
	if fake.addLimit != 500 {
		t.Errorf("DAO Add got cap %d, want 500", fake.addLimit)
	}
}

// TestAddAlreadyFavoritedIsSuccess — REQ-005 / TC-009: re-favoriting is a
// success, distinguishable from a first insert only by the flag, never by an
// error or a second row.
func TestAddAlreadyFavoritedIsSuccess(t *testing.T) {
	fake := &fakeDAO{add: func(int64, int64, int64) (dao.AddOutcome, error) {
		return dao.AddAlreadyFavorited, nil
	}}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Add(authContext(), model.URLParam{}, addRequest(10))
	if err != nil {
		t.Fatalf("re-favoriting returned an error: %v", err)
	}

	resp := payload.(svcout.AddFavoriteResponse)
	if !resp.AlreadyFavorited {
		t.Fatalf("payload = %+v, want AlreadyFavorited true", resp)
	}
}

// TestAddAtCapReturnsLimitReached — REQ-007 / TC-013: the cap is the one
// add-path failure the caller is told about, and it is a 422, not a 500.
func TestAddAtCapReturnsLimitReached(t *testing.T) {
	fake := &fakeDAO{add: func(int64, int64, int64) (dao.AddOutcome, error) {
		return dao.AddLimitReached, nil
	}}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Add(authContext(), model.URLParam{}, addRequest(99))
	if err != svcerror.ErrFavoritesLimitReached {
		t.Fatalf("Add at cap error = %v, want ErrFavoritesLimitReached", err)
	}
	if payload != nil {
		t.Fatalf("Add at cap returned payload %+v, want none", payload)
	}
}

// TestAddUnknownItemReturnsNotFound — architecture.md puts the item-existence
// check in Add; without it the insert's foreign key would be a generic 500.
func TestAddUnknownItemReturnsNotFound(t *testing.T) {
	svc := favorite.NewService(&itemMissingDAO{})

	_, _, err := svc.Add(authContext(), model.URLParam{}, addRequest(999))
	if err != svcerror.ErrCatalogItemNotFound {
		t.Fatalf("Add with unknown item error = %v, want ErrCatalogItemNotFound", err)
	}
}

// TestAddPassesUnexpectedFailuresThrough — a database failure must stay an
// unexpected error so the formator answers a generic 500. Turning it into a
// business sentinel would tell the client something false and actionable.
func TestAddPassesUnexpectedFailuresThrough(t *testing.T) {
	boom := stderrors.New("connection reset")
	fake := &fakeDAO{add: func(int64, int64, int64) (dao.AddOutcome, error) {
		return dao.AddInserted, boom
	}}
	svc := favorite.NewService(fake)

	_, _, err := svc.Add(authContext(), model.URLParam{}, addRequest(10))
	if err != boom {
		t.Fatalf("Add error = %v, want the DAO's own error back", err)
	}
}

func deleteParam(itemID string) model.URLParam {
	return model.URLParam{Path: map[string]string{favorite.PathItemID: itemID}}
}

// TestRemoveDeletesScopedToTheAuthenticatedUser — REQ-008 / TC-016: the delete
// is addressed by (token user, item), so another user's row is unreachable.
func TestRemoveDeletesScopedToTheAuthenticatedUser(t *testing.T) {
	fake := &fakeDAO{remove: func(int64, int64) (bool, error) { return true, nil }}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Remove(authContext(), deleteParam("10"), nil)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if fake.removeUser != authUserID || fake.removeItem != 10 {
		t.Errorf("DAO Remove got (%d, %d), want (%d, 10)", fake.removeUser, fake.removeItem, authUserID)
	}
	resp := payload.(svcout.RemoveFavoriteResponse)
	if resp.ItemID != 10 {
		t.Errorf("payload = %+v, want item 10", resp)
	}
}

// TestRemoveMissingRowReturnsNotFound — REQ-008 / TC-015: a row that does not
// exist is a 404, not an idempotent success.
func TestRemoveMissingRowReturnsNotFound(t *testing.T) {
	fake := &fakeDAO{remove: func(int64, int64) (bool, error) { return false, nil }}
	svc := favorite.NewService(fake)

	_, _, err := svc.Remove(authContext(), deleteParam("10"), nil)
	if err != svcerror.ErrFavoriteNotFound {
		t.Fatalf("Remove of a missing row error = %v, want ErrFavoriteNotFound", err)
	}
}

// TestRemoveNonNumericPathIsNotFound — the path segment cannot name one of the
// caller's favorites, so it gets the same 404 as any other miss rather than a
// distinguishable 400.
func TestRemoveNonNumericPathIsNotFound(t *testing.T) {
	fake := &fakeDAO{}
	svc := favorite.NewService(fake)

	for _, raw := range []string{"abc", "", "0", "-3", "1.5"} {
		_, _, err := svc.Remove(authContext(), deleteParam(raw), nil)
		if err != svcerror.ErrFavoriteNotFound {
			t.Errorf("Remove(%q) error = %v, want ErrFavoriteNotFound", raw, err)
		}
	}

	if fake.removeCalls != 0 {
		t.Errorf("a non-numeric id reached the DAO %d time(s)", fake.removeCalls)
	}
}

// TestRemovePassesUnexpectedFailuresThrough — as above, an infra failure is not
// a 404.
func TestRemovePassesUnexpectedFailuresThrough(t *testing.T) {
	boom := stderrors.New("deadlock detected")
	fake := &fakeDAO{remove: func(int64, int64) (bool, error) { return false, boom }}
	svc := favorite.NewService(fake)

	_, _, err := svc.Remove(authContext(), deleteParam("10"), nil)
	if err != boom {
		t.Fatalf("Remove error = %v, want the DAO's own error back", err)
	}
}

// TestListMapsRowsAndKeepsAnEmptyListAnArray — REQ-002 / TC-003, TC-004, and
// REQ-004's availability flag carried through from the DAO.
func TestListMapsRowsAndKeepsAnEmptyListAnArray(t *testing.T) {
	when := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	fake := &fakeDAO{list: func(int64) ([]dao.FavoriteItem, error) {
		return []dao.FavoriteItem{
			{ItemID: 10, Name: "item-10", Available: true, FavoritedAt: when},
			{ItemID: 11, Name: "item-11", Available: false, FavoritedAt: when},
		}, nil
	}}
	svc := favorite.NewService(fake)

	_, payload, err := svc.List(authContext(), model.URLParam{}, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if fake.listUserID != authUserID {
		t.Errorf("DAO List got user %d, want the token's user %d", fake.listUserID, authUserID)
	}

	list := payload.(svcout.FavoriteList)
	if len(list.Favorites) != 2 {
		t.Fatalf("payload has %d items, want 2", len(list.Favorites))
	}
	if !list.Favorites[0].Available || list.Favorites[1].Available {
		t.Errorf("availability not carried through: %+v", list.Favorites)
	}
	if !list.Favorites[0].FavoritedAt.Equal(when) {
		t.Errorf("FavoritedAt = %v, want %v", list.Favorites[0].FavoritedAt, when)
	}

	// Empty is still an array, not null.
	empty := favorite.NewService(&fakeDAO{})
	_, payload, err = empty.List(authContext(), model.URLParam{}, nil)
	if err != nil {
		t.Fatalf("List with no favorites: %v", err)
	}
	got := payload.(svcout.FavoriteList)
	if got.Favorites == nil {
		t.Fatal("an empty list produced a nil slice, which serialises as null")
	}
	if len(got.Favorites) != 0 {
		t.Fatalf("expected no items, got %d", len(got.Favorites))
	}
}

// TestListPassesUnexpectedFailuresThrough — a failed list is a 500, not an
// empty favorites page. Silently returning [] here would look like REQ-002's
// legitimate empty state and hide a real outage.
func TestListPassesUnexpectedFailuresThrough(t *testing.T) {
	boom := stderrors.New("query timeout")
	fake := &fakeDAO{list: func(int64) ([]dao.FavoriteItem, error) { return nil, boom }}
	svc := favorite.NewService(fake)

	_, _, err := svc.List(authContext(), model.URLParam{}, nil)
	if err != boom {
		t.Fatalf("List error = %v, want the DAO's own error back", err)
	}
}

// The service must satisfy nexcommon's services.Services contract; a compile
// failure here is the assertion.
var _ interface {
	GetDTO() interface{}
	GetMultipartDTO() interface{}
	GetListScope() []string
} = (*favorite.Service)(nil)

// The fake must satisfy the DAO interface the service is built against.
var _ dao.FavoriteDAO = (*fakeDAO)(nil)
