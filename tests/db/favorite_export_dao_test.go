// FEAT-003's database-backed tests: the export query's join, ordering, price
// canonicalization and nullable note.
//
// These assert real SQL semantics — the live join, trim_scale, and the
// LEFT JOIN keeping a discontinued item — which a fake would only echo back.
// TestMain skips the package when docker is unavailable.
package db

import (
	"testing"

	"github.com/william22913/sandbox-svc-code/tests/testdb"
)

// TestListForExportReadsLiveValuesAndNullableNote — REQ-003, REQ-005, REQ-015,
// TC-010, TC-011, TC-016, TC-017, TC-032.
func TestListForExportReadsLiveValuesAndNullableNote(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)

	exec(t, `INSERT INTO catalog_items (id, name, category, price, is_available) VALUES
		(10, 'Notebook, A5', 'Office', 19.99, TRUE),
		(11, 'Pen',          'Office', 1500,  FALSE)`)
	exec(t, `INSERT INTO favorite_items (user_id, item_id, created_at, note) VALUES
		(1, 10, '2026-04-09T14:30:00Z', NULL),
		(1, 11, '2026-04-10T00:00:00Z', 'Café — 🎁')`)

	items, err := favorite.ListForExport(1)
	if err != nil {
		t.Fatalf("ListForExport: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d rows, want 2 — a discontinued item must not be dropped", len(items))
	}

	// Newest-favorited first, exactly like List.
	if items[0].ItemID != 11 || items[1].ItemID != 10 {
		t.Fatalf("order = [%d %d], want [11 10]", items[0].ItemID, items[1].ItemID)
	}

	if items[0].Price != "1500" {
		t.Errorf("integer price = %q, want 1500 with no trailing scale", items[0].Price)
	}
	if items[1].Price != "19.99" {
		t.Errorf("fractional price = %q, want 19.99", items[1].Price)
	}
	if items[1].Category != "Office" {
		t.Errorf("category = %q, want Office", items[1].Category)
	}
	if items[0].Available {
		t.Error("discontinued item reported available")
	}
	if items[1].Note.Valid {
		t.Errorf("NULL note scanned as valid %q", items[1].Note.String)
	}
	if !items[0].Note.Valid || items[0].Note.String != "Café — 🎁" {
		t.Errorf("note = %+v, want the stored string", items[0].Note)
	}
	if got := items[1].FavoritedAt.UTC().Format("2006-01-02"); got != "2026-04-09" {
		t.Errorf("favorited_at = %q, want 2026-04-09", got)
	}

	// REQ-015 / TC-032: read live, not a snapshot.
	exec(t, `UPDATE catalog_items SET name = 'New name', price = 12.5, is_available = FALSE WHERE id = 10`)

	items, err = favorite.ListForExport(1)
	if err != nil {
		t.Fatalf("ListForExport after catalog update: %v", err)
	}
	if items[1].Name != "New name" || items[1].Price != "12.5" || items[1].Available {
		t.Errorf("row after catalog update = %+v, want the new name/price/availability", items[1])
	}
}

// TestListForExportCanonicalizesPriceScale — REQ-005, TC-011, TC-012: the
// stored scale cannot leak trailing zeros into the CSV.
func TestListForExportCanonicalizesPriceScale(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	exec(t, `INSERT INTO catalog_items (id, name, category, price, is_available) VALUES
		(10, 'a', '', 19.9900,  TRUE),
		(11, 'b', '', 1234.5678, TRUE)`)
	exec(t, `INSERT INTO favorite_items (user_id, item_id, created_at) VALUES
		(1, 10, '2026-04-09T00:00:00Z'),
		(1, 11, '2026-04-10T00:00:00Z')`)

	items, err := favorite.ListForExport(1)
	if err != nil {
		t.Fatalf("ListForExport: %v", err)
	}

	if items[0].Price != "1234.5678" {
		t.Errorf("price = %q, want 1234.5678 unrounded", items[0].Price)
	}
	if items[1].Price != "19.99" {
		t.Errorf("price = %q, want 19.99 without the stored trailing zeros", items[1].Price)
	}
}

// TestListForExportScopesToTheUser — REQ-013, TC-030, TC-031.
func TestListForExportScopesToTheUser(t *testing.T) {
	testdb.Truncate(t, db)
	seedUser(t, 1)
	seedUser(t, 2)
	seedItem(t, 10, true)
	seedItem(t, 11, true)

	exec(t, `INSERT INTO favorite_items (user_id, item_id) VALUES (1, 10), (2, 11)`)

	items, err := favorite.ListForExport(1)
	if err != nil {
		t.Fatalf("ListForExport: %v", err)
	}
	if len(items) != 1 || items[0].ItemID != 10 {
		t.Fatalf("user 1 export = %+v, want only item 10", items)
	}
}
