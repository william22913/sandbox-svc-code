// Package db holds the database-backed tests. It is a separate package from
// tests/ so its TestMain — which owns the Postgres container — cannot skip the
// bundle guard tests when docker is absent.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/tests/testdb"
)

var (
	db       *sql.DB
	cleanup  func()
	favorite dao.FavoriteDAO
)

func TestMain(m *testing.M) {
	var err error
	db, cleanup, err = testdb.Start()
	if errors.Is(err, testdb.ErrNoDocker) {
		fmt.Println("SKIP: docker not available — database-backed tests did not run")
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting test database: %v\n", err)
		os.Exit(1)
	}
	favorite = dao.NewFavoriteDAO(db)

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func seedUser(t *testing.T, id int64) {
	t.Helper()
	exec(t, `INSERT INTO users (id, username) VALUES ($1, $2)`, id, fmt.Sprintf("user-%d", id))
}

func seedItem(t *testing.T, id int64, available bool) {
	t.Helper()
	exec(t, `INSERT INTO catalog_items (id, name, is_available) VALUES ($1, $2, $3)`,
		id, fmt.Sprintf("item-%d", id), available)
}

func exec(t *testing.T, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func countFavorites(t *testing.T, userID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM favorite_items WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("counting favorites: %v", err)
	}
	return n
}

// TestAddInsertsThenIsIdempotent — REQ-005 / TC-009.
func TestAddInsertsThenIsIdempotent(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedItem(t, 10, true)

	outcome, err := favorite.Add(1, 10, 500)
	if err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if outcome != dao.AddInserted {
		t.Fatalf("first Add outcome = %v, want AddInserted", outcome)
	}

	outcome, err = favorite.Add(1, 10, 500)
	if err != nil {
		t.Fatalf("second Add: %v", err)
	}
	if outcome != dao.AddAlreadyFavorited {
		t.Fatalf("second Add outcome = %v, want AddAlreadyFavorited", outcome)
	}

	if got := countFavorites(t, 1); got != 1 {
		t.Fatalf("favorite count = %d, want 1 (no duplicate row)", got)
	}
}

// TestAddAtCapFailsWithoutInsertingOrEvicting — REQ-007 / TC-012, TC-013.
func TestAddAtCapFailsWithoutInsertingOrEvicting(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	const limit = 3
	for i := int64(10); i < 10+limit; i++ {
		seedItem(t, i, true)
		if _, err := favorite.Add(1, i, limit); err != nil {
			t.Fatalf("seeding favorite %d: %v", i, err)
		}
	}
	if got := countFavorites(t, 1); got != limit {
		t.Fatalf("setup: favorite count = %d, want %d", got, limit)
	}

	seedItem(t, 99, true)
	outcome, err := favorite.Add(1, 99, limit)
	if err != nil {
		t.Fatalf("Add at cap: %v", err)
	}
	if outcome != dao.AddLimitReached {
		t.Fatalf("Add at cap outcome = %v, want AddLimitReached", outcome)
	}

	// No silent insert, and no eviction of an existing favorite to make room.
	if got := countFavorites(t, 1); got != limit {
		t.Fatalf("favorite count after cap rejection = %d, want %d", got, limit)
	}
	var overflow int
	if err := db.QueryRow(`SELECT count(*) FROM favorite_items WHERE user_id = 1 AND item_id = 99`).Scan(&overflow); err != nil {
		t.Fatalf("checking rejected item: %v", err)
	}
	if overflow != 0 {
		t.Fatal("the rejected item was inserted anyway")
	}
}

// TestRefavoritingAtCapStillSucceeds — REQ-005 and REQ-007 together: the cap is
// on distinct favorites, so re-favoriting something already in the list must not
// report the cap. This is why the existence check runs before the count.
func TestRefavoritingAtCapStillSucceeds(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	const limit = 3
	for i := int64(10); i < 10+limit; i++ {
		seedItem(t, i, true)
		if _, err := favorite.Add(1, i, limit); err != nil {
			t.Fatalf("seeding favorite %d: %v", i, err)
		}
	}

	outcome, err := favorite.Add(1, 10, limit)
	if err != nil {
		t.Fatalf("re-favoriting at cap: %v", err)
	}
	if outcome != dao.AddAlreadyFavorited {
		t.Fatalf("re-favoriting at cap outcome = %v, want AddAlreadyFavorited", outcome)
	}
}

// TestRemoveOnlyTouchesTheCallersOwnRow — REQ-008 / TC-015, TC-016.
func TestRemoveOnlyTouchesTheCallersOwnRow(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedUser(t, 2)
	seedItem(t, 10, true)

	if _, err := favorite.Add(1, 10, 500); err != nil {
		t.Fatalf("seed user 1 favorite: %v", err)
	}
	if _, err := favorite.Add(2, 10, 500); err != nil {
		t.Fatalf("seed user 2 favorite: %v", err)
	}

	// User 2 removes their own row.
	removed, err := favorite.Remove(2, 10)
	if err != nil {
		t.Fatalf("Remove as user 2: %v", err)
	}
	if !removed {
		t.Fatal("Remove on the caller's own row reported nothing removed")
	}

	// Removing it again is "no such row" — the same answer as someone else's row.
	removed, err = favorite.Remove(2, 10)
	if err != nil {
		t.Fatalf("second Remove as user 2: %v", err)
	}
	if removed {
		t.Fatal("Remove on a now-missing row reported a removal")
	}

	// User 2's delete must not have touched user 1's identical (item) row.
	if got := countFavorites(t, 1); got != 1 {
		t.Fatalf("user 1 favorite count = %d, want 1 — another user's delete removed it", got)
	}

	// A cross-user attempt behaves exactly like a missing row: false, no error.
	removed, err = favorite.Remove(3, 10)
	if err != nil {
		t.Fatalf("Remove as unknown user: %v", err)
	}
	if removed {
		t.Fatal("Remove as a user with no such row reported a removal")
	}
	if got := countFavorites(t, 1); got != 1 {
		t.Fatalf("user 1 favorite count = %d after cross-user remove, want 1", got)
	}
}

// TestListReportsDiscontinuedItemAsUnavailable — REQ-004 / TC-007, TC-008. The
// item is favorited while available and discontinued afterwards; the list reads
// the flag live, so it still appears, marked unavailable.
func TestListReportsDiscontinuedItemAsUnavailable(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedItem(t, 10, true)
	seedItem(t, 11, true)

	if _, err := favorite.Add(1, 10, 500); err != nil {
		t.Fatalf("favorite available item: %v", err)
	}
	if _, err := favorite.Add(1, 11, 500); err != nil {
		t.Fatalf("favorite to-be-discontinued item: %v", err)
	}

	// Discontinued after favoriting — a soft mark, never a row delete.
	exec(t, `UPDATE catalog_items SET is_available = FALSE WHERE id = 11`)

	items, err := favorite.List(1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List returned %d items, want 2 (a discontinued item must not vanish)", len(items))
	}

	byID := map[int64]dao.FavoriteItem{}
	for _, item := range items {
		byID[item.ItemID] = item
	}
	if got := byID[10]; !got.Available || got.Name != "item-10" {
		t.Errorf("item 10 = %+v, want available with its catalog name", got)
	}
	if got := byID[11]; got.Available {
		t.Errorf("item 11 = %+v, want Available false", got)
	}
	if got := byID[11]; got.Name != "item-11" {
		t.Errorf("item 11 name = %q, want the catalog name even when unavailable", got.Name)
	}
}

// TestRemovingADiscontinuedItemStillWorks — REQ-004 / TC-008. The remove is
// keyed on the favorite_items row alone, so a discontinued item is removable
// like any other; nothing about it consults the catalog.
func TestRemovingADiscontinuedItemStillWorks(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedItem(t, 10, true)

	if _, err := favorite.Add(1, 10, 500); err != nil {
		t.Fatalf("favorite the item: %v", err)
	}
	exec(t, `UPDATE catalog_items SET is_available = FALSE WHERE id = 10`)

	removed, err := favorite.Remove(1, 10)
	if err != nil {
		t.Fatalf("Remove a discontinued favorite: %v", err)
	}
	if !removed {
		t.Fatal("Remove reported nothing removed for a discontinued favorite")
	}
	if got := countFavorites(t, 1); got != 0 {
		t.Fatalf("favorite count = %d, want 0", got)
	}
}

// TestUnfavoritingAtCapFreesASlot — REQ-007 with REQ-003 / TC-014. The cap is
// on the live count, not on a lifetime total, so dropping one favorite must let
// the next distinct one in.
func TestUnfavoritingAtCapFreesASlot(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	const limit = 3
	for i := int64(10); i < 10+limit; i++ {
		seedItem(t, i, true)
		if _, err := favorite.Add(1, i, limit); err != nil {
			t.Fatalf("seeding favorite %d: %v", i, err)
		}
	}

	seedItem(t, 99, true)
	if outcome, err := favorite.Add(1, 99, limit); err != nil || outcome != dao.AddLimitReached {
		t.Fatalf("setup: Add at cap = (%v, %v), want AddLimitReached", outcome, err)
	}

	removed, err := favorite.Remove(1, 10)
	if err != nil {
		t.Fatalf("Remove to free a slot: %v", err)
	}
	if !removed {
		t.Fatal("Remove reported nothing removed")
	}

	outcome, err := favorite.Add(1, 99, limit)
	if err != nil {
		t.Fatalf("Add after freeing a slot: %v", err)
	}
	if outcome != dao.AddInserted {
		t.Fatalf("Add after freeing a slot outcome = %v, want AddInserted", outcome)
	}
	if got := countFavorites(t, 1); got != limit {
		t.Fatalf("favorite count = %d, want %d", got, limit)
	}
}

// TestListIsEmptyNotNilWithoutFavorites — TC-004. An empty list must serialise
// as [] rather than null.
func TestListIsEmptyNotNilWithoutFavorites(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)

	items, err := favorite.List(1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if items == nil {
		t.Fatal("List returned nil, want an empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("List returned %d items for a user with no favorites", len(items))
	}
}

// seedBulkFavorites seeds count favorites for userID over item ids from base,
// the last `discontinued` of them unavailable. Bulk SQL rather than Add loops:
// TC-006 needs 500 rows and the cap policy is FEAT-001's concern, already
// tested above. Callers pass distinct bases so two users never share item rows.
func seedBulkFavorites(t *testing.T, userID, base, count, discontinued int64) {
	t.Helper()
	available := count - discontinued

	if _, err := db.Exec(
		`INSERT INTO catalog_items (id, name, is_available)
		 SELECT g, 'item-' || g, g < $1
		   FROM generate_series($2::bigint, $3::bigint) g`,
		base+available, base, base+count-1,
	); err != nil {
		t.Fatalf("bulk-seeding catalog items: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO favorite_items (user_id, item_id)
		 SELECT $1, g FROM generate_series($2::bigint, $3::bigint) g`,
		userID, base, base+count-1,
	); err != nil {
		t.Fatalf("bulk-seeding favorites: %v", err)
	}
}

// TestSummary_TC001_CountsTotalAndDiscontinued — REQ-003, REQ-004: one
// aggregate over the user's rows, with availability as a filter on the
// discontinued count only.
func TestSummary_TC001_CountsTotalAndDiscontinued(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedItem(t, 10, true)
	seedItem(t, 11, true)
	seedItem(t, 12, false)
	for _, itemID := range []int64{10, 11, 12} {
		if _, err := favorite.Add(1, itemID, 500); err != nil {
			t.Fatalf("seeding favorite %d: %v", itemID, err)
		}
	}

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 3 || got.Discontinued != 1 {
		t.Fatalf("summary = %+v, want {Total:3 Discontinued:1}", got)
	}
}

// TestSummary_TC002_ZeroFavorites — REQ-012: COUNT over no rows is 0/0, not an
// error and not a missing row.
func TestSummary_TC002_ZeroFavorites(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 0 || got.Discontinued != 0 {
		t.Fatalf("summary = %+v, want {Total:0 Discontinued:0}", got)
	}
}

// TestSummary_TC003_SingleAvailableFavorite — the lower non-zero boundary.
func TestSummary_TC003_SingleAvailableFavorite(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedItem(t, 10, true)
	if _, err := favorite.Add(1, 10, 500); err != nil {
		t.Fatalf("seeding favorite: %v", err)
	}

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 1 || got.Discontinued != 0 {
		t.Fatalf("summary = %+v, want {Total:1 Discontinued:0}", got)
	}
}

// TestSummary_TC004_MixedPartition — 5 rows, 2 discontinued: total is not
// availability-filtered and discontinued is exactly the marked subset.
func TestSummary_TC004_MixedPartition(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 5, 2)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 5 || got.Discontinued != 2 {
		t.Fatalf("summary = %+v, want {Total:5 Discontinued:2}", got)
	}
}

// TestSummary_TC005_AllDiscontinuedEqual — REQ-014: no row leaves the total.
func TestSummary_TC005_AllDiscontinuedEqual(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 3, 3)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 3 || got.Discontinued != 3 {
		t.Fatalf("summary = %+v, want {Total:3 Discontinued:3}", got)
	}
}

// TestSummary_TC006_FiveHundredCap — REQ-009 at the FEAT-001 maximum.
func TestSummary_TC006_FiveHundredCap(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 500, 125)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 500 || got.Discontinued != 125 {
		t.Fatalf("summary = %+v, want {Total:500 Discontinued:125}", got)
	}
}

// TestSummary_TC007_AboveNinetyNineNotCapped — the badge's 99+ rule is not in
// the query: 100 rows come back as 100.
func TestSummary_TC007_AboveNinetyNineNotCapped(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 100, 12)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 100 || got.Discontinued != 12 {
		t.Fatalf("summary = %+v, want {Total:100 Discontinued:12}", got)
	}
}

// TestSummary_TC008_SingleDiscontinued — the minimum equal non-zero case.
func TestSummary_TC008_SingleDiscontinued(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 1, 1)

	got, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Total != 1 || got.Discontinued != 1 {
		t.Fatalf("summary = %+v, want {Total:1 Discontinued:1}", got)
	}
}

// TestSummary_TC011_IsolatedPerUser — REQ-010: the WHERE clause is the whole
// isolation guarantee, so disjoint users must get disjoint answers.
func TestSummary_TC011_IsolatedPerUser(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedUser(t, 2)
	seedBulkFavorites(t, 1, 5000, 2, 0)

	// User 2's rows use a different item range so neither set is shared.
	seedBulkFavorites(t, 2, 6000, 6, 4)

	gotA, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("Summary as user 1: %v", err)
	}
	gotB, err := favorite.Summary(2)
	if err != nil {
		t.Fatalf("Summary as user 2: %v", err)
	}

	if gotA.Total != 2 || gotA.Discontinued != 0 {
		t.Errorf("user 1 summary = %+v, want {2,0}", gotA)
	}
	if gotB.Total != 6 || gotB.Discontinued != 4 {
		t.Errorf("user 2 summary = %+v, want {6,4}", gotB)
	}
}

// TestSummary_TC014_ReflectsCatalogChangesAtRequestTime — REQ-009: the count is
// derived per request, so a soft-mark between calls changes only the
// discontinued figure and no favorite row moves.
func TestSummary_TC014_ReflectsCatalogChangesAtRequestTime(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedBulkFavorites(t, 1, 5000, 3, 0)

	first, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("first Summary: %v", err)
	}
	if first.Total != 3 || first.Discontinued != 0 {
		t.Fatalf("first summary = %+v, want {3,0}", first)
	}

	exec(t, `UPDATE catalog_items SET is_available = FALSE WHERE id = 5000`)

	second, err := favorite.Summary(1)
	if err != nil {
		t.Fatalf("second Summary: %v", err)
	}
	if second.Total != 3 || second.Discontinued != 1 {
		t.Fatalf("second summary = %+v, want {3,1} — the change was not read live", second)
	}
}

// TestAddCapHoldsUnderConcurrentRequests — REQ-007's real guarantee. Without the
// per-user row lock in Add, the count-then-insert pre-check lets concurrent
// requests each read the same count and all insert, pushing the user past the
// cap. Run with -race as well.
func TestAddCapHoldsUnderConcurrentRequests(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)

	const (
		limit   = 5
		preSeed = limit - 1
		extra   = 8
	)
	for i := int64(10); i < 10+preSeed; i++ {
		seedItem(t, i, true)
		if _, err := favorite.Add(1, i, limit); err != nil {
			t.Fatalf("seeding favorite %d: %v", i, err)
		}
	}
	// Distinct items, so they are genuinely competing for the last slot.
	for i := int64(100); i < 100+extra; i++ {
		seedItem(t, i, true)
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		inserted int
		rejected int
		failures []error
	)

	start := make(chan struct{})
	for i := int64(100); i < 100+extra; i++ {
		wg.Add(1)
		go func(itemID int64) {
			defer wg.Done()
			<-start
			outcome, err := favorite.Add(1, itemID, limit)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				failures = append(failures, err)
			case outcome == dao.AddInserted:
				inserted++
			case outcome == dao.AddLimitReached:
				rejected++
			default:
				failures = append(failures, fmt.Errorf("unexpected outcome %v for item %d", outcome, itemID))
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if len(failures) > 0 {
		t.Fatalf("concurrent Add errors: %v", failures)
	}
	if inserted != 1 {
		t.Errorf("%d inserts succeeded, want exactly 1 (one free slot)", inserted)
	}
	if rejected != extra-1 {
		t.Errorf("%d requests were rejected, want %d", rejected, extra-1)
	}
	if got := countFavorites(t, 1); got != limit {
		t.Errorf("final favorite count = %d, want %d — the cap was exceeded", got, limit)
	}
}
