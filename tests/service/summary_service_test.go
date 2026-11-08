package service

import (
	stderrors "errors"
	"testing"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"

	"github.com/william22913/sandbox-svc-code/dao"
	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// summaryContext builds a request context whose token names userID — the only
// source of identity the summary service may consult.
func summaryContext(userID int64) *internalCtx.ContextModel {
	ctx := internalCtx.NewContextModel()
	ctx.AuthAccessTokenModel.AuthenticationServerUserID = userID
	return ctx
}

// summaryFor runs Summary against a fake returning the given aggregate and
// returns the typed payload, failing if the service answered an error.
func summaryFor(t *testing.T, userID, total, discontinued int64) svcout.FavoriteSummaryResponse {
	t.Helper()

	fake := &fakeDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{Total: total, Discontinued: discontinued}, nil
	}}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Summary(summaryContext(userID), model.URLParam{}, nil)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	resp, ok := payload.(svcout.FavoriteSummaryResponse)
	if !ok {
		t.Fatalf("payload type = %T, want out.FavoriteSummaryResponse", payload)
	}
	return resp
}

// TestSummary_TC001_MixedCountsAreMapped — REQ-001..REQ-004: the aggregate the
// DAO returns is carried through unchanged, both fields populated.
func TestSummary_TC001_MixedCountsAreMapped(t *testing.T) {
	resp := summaryFor(t, authUserID, 3, 1)
	if resp.Total != 3 || resp.Discontinued != 1 {
		t.Fatalf("payload = %+v, want {Total:3 Discontinued:1}", resp)
	}
}

// TestSummary_TC002_ZeroFavorites — REQ-012: no favorites is a real answer,
// not an error and not a nil payload.
func TestSummary_TC002_ZeroFavorites(t *testing.T) {
	resp := summaryFor(t, authUserID, 0, 0)
	if resp.Total != 0 || resp.Discontinued != 0 {
		t.Fatalf("payload = %+v, want {Total:0 Discontinued:0}", resp)
	}
}

// TestSummary_TC003_SingleAvailableFavorite — the lower non-zero boundary.
func TestSummary_TC003_SingleAvailableFavorite(t *testing.T) {
	resp := summaryFor(t, authUserID, 1, 0)
	if resp.Total != 1 || resp.Discontinued != 0 {
		t.Fatalf("payload = %+v, want {Total:1 Discontinued:0}", resp)
	}
}

// TestSummary_TC004_MixedPartition — total is not filtered by availability;
// discontinued is the subset.
func TestSummary_TC004_MixedPartition(t *testing.T) {
	resp := summaryFor(t, authUserID, 5, 2)
	if resp.Total != 5 || resp.Discontinued != 2 {
		t.Fatalf("payload = %+v, want {Total:5 Discontinued:2}", resp)
	}
}

// TestSummary_TC005_AllDiscontinuedEqual — REQ-014: equal counts survive the
// mapping, which would break if either field were derived from the other.
func TestSummary_TC005_AllDiscontinuedEqual(t *testing.T) {
	resp := summaryFor(t, authUserID, 3, 3)
	if resp.Total != 3 || resp.Discontinued != 3 {
		t.Fatalf("payload = %+v, want {Total:3 Discontinued:3}", resp)
	}
}

// TestSummary_TC006_FiveHundredCap — REQ-009 at FEAT-001's maximum: no
// off-by-one or truncation.
func TestSummary_TC006_FiveHundredCap(t *testing.T) {
	resp := summaryFor(t, authUserID, 500, 125)
	if resp.Total != 500 || resp.Discontinued != 125 {
		t.Fatalf("payload = %+v, want {Total:500 Discontinued:125}", resp)
	}
}

// TestSummary_TC007_AboveNinetyNineNotCapped — REQ-005: the 99+ rule is a badge
// display cap, never a value returned by the API.
func TestSummary_TC007_AboveNinetyNineNotCapped(t *testing.T) {
	resp := summaryFor(t, authUserID, 100, 12)
	if resp.Total != 100 || resp.Discontinued != 12 {
		t.Fatalf("payload = %+v, want {Total:100 Discontinued:12}", resp)
	}
}

// TestSummary_TC008_SingleDiscontinued — the minimum case where both counts are
// equal and non-zero.
func TestSummary_TC008_SingleDiscontinued(t *testing.T) {
	resp := summaryFor(t, authUserID, 1, 1)
	if resp.Total != 1 || resp.Discontinued != 1 {
		t.Fatalf("payload = %+v, want {Total:1 Discontinued:1}", resp)
	}
}

// TestSummary_TC011_IsolatedPerUser — REQ-010: the service asks for the user in
// the token, so two tokens get two different answers and neither borrows the
// other's rows.
func TestSummary_TC011_IsolatedPerUser(t *testing.T) {
	const otherUserID int64 = 77

	rows := map[int64]dao.FavoriteSummary{
		authUserID:  {Total: 2, Discontinued: 0},
		otherUserID: {Total: 6, Discontinued: 4},
	}
	fake := &fakeDAO{summary: func(userID int64) (dao.FavoriteSummary, error) {
		return rows[userID], nil
	}}
	svc := favorite.NewService(fake)

	for userID, want := range rows {
		_, payload, err := svc.Summary(summaryContext(userID), model.URLParam{}, nil)
		if err != nil {
			t.Fatalf("Summary as user %d: %v", userID, err)
		}
		if got := payload.(svcout.FavoriteSummaryResponse); got.Total != want.Total || got.Discontinued != want.Discontinued {
			t.Errorf("user %d payload = %+v, want %+v", userID, got, want)
		}
	}

	if fake.summaryCalls != len(rows) {
		t.Errorf("DAO Summary called %d times, want one per request (%d)", fake.summaryCalls, len(rows))
	}
}

// TestSummary_TC012_CallerInputIsIgnored — REQ-011 / REQ-026: query-ish path
// params and a body are inert; only the token's user reaches the DAO.
func TestSummary_TC012_CallerInputIsIgnored(t *testing.T) {
	fake := &fakeDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{Total: 2, Discontinued: 0}, nil
	}}
	svc := favorite.NewService(fake)

	// The URL the caller asked for and a body claiming different values.
	param := model.URLParam{Path: map[string]string{"user_id": "77"}}
	body := map[string]interface{}{"user_id": 77, "total": 999, "discontinued": 999}

	_, payload, err := svc.Summary(summaryContext(authUserID), param, body)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if fake.summaryUser != authUserID {
		t.Errorf("DAO Summary got user %d, want the token's user %d", fake.summaryUser, authUserID)
	}
	if got := payload.(svcout.FavoriteSummaryResponse); got.Total != 2 || got.Discontinued != 0 {
		t.Errorf("payload = %+v, want the token user's own {2,0}", got)
	}
}

// TestSummary_TC013_ExtraQueryParametersCannotOverride — REQ-011: the route has
// no parameters to bind, so arbitrary extras change nothing.
func TestSummary_TC013_ExtraQueryParametersCannotOverride(t *testing.T) {
	fake := &fakeDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{Total: 4, Discontinued: 1}, nil
	}}
	svc := favorite.NewService(fake)

	param := model.URLParam{Query: map[string]string{
		"total":        "0",
		"discontinued": "0",
		"limit":        "1000",
		"offset":       "0",
	}}

	_, payload, err := svc.Summary(summaryContext(authUserID), param, nil)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got := payload.(svcout.FavoriteSummaryResponse); got.Total != 4 || got.Discontinued != 1 {
		t.Fatalf("payload = %+v, want the derived {4,1}", got)
	}
}

// TestSummary_TC014_DerivedPerRequest — REQ-009: nothing is stored above the
// DAO, so the second call sees the second aggregate. A cached response would
// repeat the first.
func TestSummary_TC014_DerivedPerRequest(t *testing.T) {
	var unavailable bool
	fake := &fakeDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		if unavailable {
			return dao.FavoriteSummary{Total: 3, Discontinued: 1}, nil
		}
		return dao.FavoriteSummary{Total: 3, Discontinued: 0}, nil
	}}
	svc := favorite.NewService(fake)

	_, first, err := svc.Summary(summaryContext(authUserID), model.URLParam{}, nil)
	if err != nil {
		t.Fatalf("first Summary: %v", err)
	}
	if got := first.(svcout.FavoriteSummaryResponse); got.Discontinued != 0 {
		t.Fatalf("first payload = %+v, want discontinued 0", got)
	}

	// The catalog item is soft-marked discontinued between requests; no
	// favorite row changes.
	unavailable = true

	_, second, err := svc.Summary(summaryContext(authUserID), model.URLParam{}, nil)
	if err != nil {
		t.Fatalf("second Summary: %v", err)
	}
	if got := second.(svcout.FavoriteSummaryResponse); got.Total != 3 || got.Discontinued != 1 {
		t.Fatalf("second payload = %+v, want {3,1} from a fresh read", got)
	}
	if fake.summaryCalls != 2 {
		t.Errorf("DAO Summary called %d times, want 2 — a stored count would skip the read", fake.summaryCalls)
	}
}

// TestSummary_TC015_UnexpectedFailurePassesThrough — REQ-017: an infra error
// stays an error, so the formator answers a 500 instead of a false 0/0.
func TestSummary_TC015_UnexpectedFailurePassesThrough(t *testing.T) {
	boom := stderrors.New("catalog availability lookup failed")
	fake := &fakeDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{}, boom
	}}
	svc := favorite.NewService(fake)

	_, payload, err := svc.Summary(summaryContext(authUserID), model.URLParam{}, nil)
	if err != boom {
		t.Fatalf("Summary error = %v, want the DAO's own error back", err)
	}
	if payload != nil {
		t.Fatalf("Summary returned payload %+v alongside the error", payload)
	}
}
