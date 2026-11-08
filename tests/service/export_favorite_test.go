// FEAT-003 service tests: the CSV renderer, exercised through its only public
// entry point against a fake DAO.
//
// Every assertion here is about the file's bytes or its decoded cells — the
// header, ordering, price/date/availability formatting, NULL notes, RFC 4180
// quoting, the BOM/CRLF framing and formula neutralization. The SQL itself is
// covered separately by tests/db against a real Postgres.
package service

import (
	"database/sql"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// csvHeader is the exact expected prologue: BOM, header row, CRLF.
const csvHeader = "\xef\xbb\xbfitem_id,name,category,price,available,favorited_at,note\r\n"

var exportWhen = time.Date(2026, 4, 9, 14, 30, 0, 0, time.UTC)

func nullNote(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

// exportBody renders the rows as the given user and fails the test on error.
func exportBody(t *testing.T, rows []dao.FavoriteExportItem) string {
	t.Helper()

	fake := &fakeDAO{listExport: func(int64) ([]dao.FavoriteExportItem, error) {
		return rows, nil
	}}

	body, err := favorite.NewService(fake).ExportCSV(authContext())
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}

	return string(body)
}

// TestExportCSVHeaderBOMAndCRLF_TC004_TC020 — the file opens with exactly one
// UTF-8 BOM, then REQ-004's header, and every record ends in CRLF. No bare LF
// appears because no field in these rows contains one.
func TestExportCSVHeaderBOMAndCRLF_TC004_TC020(t *testing.T) {
	body := exportBody(t, nil)

	if !strings.HasPrefix(body, csvHeader) {
		t.Fatalf("body starts with %q, want %q", firstBytes(body, len(csvHeader)), csvHeader)
	}
	if strings.Count(body, "\xef\xbb\xbf") != 1 {
		t.Errorf("BOM appears %d times, want exactly 1", strings.Count(body, "\xef\xbb\xbf"))
	}
	if strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") {
		t.Error("body contains a bare LF, want CRLF record terminators only")
	}
	if !strings.HasSuffix(body, "\r\n") {
		t.Error("body does not end with CRLF")
	}
}

// TestExportCSVKeepsDAOOrder_TC005_TC008 — the file's row order is the DAO's
// order (newest first, ties broken by item id), never re-sorted here.
func TestExportCSVKeepsDAOOrder_TC005_TC008(t *testing.T) {
	rows := []dao.FavoriteExportItem{
		{ItemID: 30, Name: "C", Category: "cat-c", Price: "30", Available: true, FavoritedAt: exportWhen, Note: nullNote("c")},
		{ItemID: 20, Name: "B", Category: "cat-b", Price: "20", Available: true, FavoritedAt: exportWhen, Note: nullNote("b")},
		{ItemID: 10, Name: "A", Category: "cat-a", Price: "10", Available: true, FavoritedAt: exportWhen, Note: nullNote("a")},
	}

	of := parseCSV(t, exportBody(t, rows))
	if len(of) != 4 {
		t.Fatalf("parsed %d records, want 1 header + 3 rows", len(of))
	}

	for i, wantID := range []string{"30", "20", "10"} {
		if got := of[i+1][0]; got != wantID {
			t.Errorf("row %d item_id = %q, want %q — DAO order was not preserved", i+1, got, wantID)
		}
	}
}

// TestExportCSVRendersEveryRow_TC006_TC007 — the smallest non-empty export and
// the 500-row cap both render completely: no paging, no truncation.
func TestExportCSVRendersEveryRow_TC006_TC007(t *testing.T) {
	one := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 10, Name: "Only", Category: "cat", Price: "1", Available: true, FavoritedAt: exportWhen},
	})
	if got := len(parseCSV(t, one)); got != 2 {
		t.Fatalf("one-favorite export has %d records, want 1 header + 1 row", got)
	}

	rows := make([]dao.FavoriteExportItem, 0, favorite.MaxFavoritesPerUser)
	for i := int64(0); i < favorite.MaxFavoritesPerUser; i++ {
		rows = append(rows, dao.FavoriteExportItem{
			ItemID:      i + 1,
			Name:        "item",
			Category:    "cat",
			Price:       "1",
			Available:   true,
			FavoritedAt: exportWhen,
		})
	}

	body := exportBody(t, rows)
	if got := strings.Count(body, "\r\n"); got != int(favorite.MaxFavoritesPerUser)+1 {
		t.Fatalf("500-favorite export has %d records, want 501", got)
	}
}

// TestExportCSVPriceVerbatim_TC010_TC011_TC012 — price is written exactly as
// the query returned it: no symbol, no grouping, no rounding, no added scale.
func TestExportCSVPriceVerbatim_TC010_TC011_TC012(t *testing.T) {
	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "i", Category: "c", Price: "1500", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: "i", Category: "c", Price: "19.99", Available: true, FavoritedAt: exportWhen},
		{ItemID: 3, Name: "i", Category: "c", Price: "1234.5678", Available: true, FavoritedAt: exportWhen},
	})

	of := parseCSV(t, body)
	for i, want := range []string{"1500", "19.99", "1234.5678"} {
		if got := of[i+1][3]; got != want {
			t.Errorf("row %d price = %q, want %q", i+1, got, want)
		}
	}
}

// TestExportCSVDateIsUTC_TC013_TC014 — favorited_at is the UTC calendar date.
// The second row is 02:30 on 10 April in Jakarta, which is still 9 April in
// UTC; formatting the server-local value would emit 2026-04-10.
func TestExportCSVDateIsUTC_TC013_TC014(t *testing.T) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)

	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: time.Date(2026, 4, 10, 2, 30, 0, 0, jakarta)},
		{ItemID: 3, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)},
		{ItemID: 4, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	})

	of := parseCSV(t, body)
	for i, want := range []string{"2026-04-09", "2026-04-09", "2026-02-28", "2026-03-01"} {
		if got := of[i+1][5]; got != want {
			t.Errorf("row %d favorited_at = %q, want %q", i+1, got, want)
		}
	}
}

// TestExportCSVAvailabilityLabels_TC015_TC016 — the two exact Indonesian
// labels, including for a discontinued favorite that is still exported.
func TestExportCSVAvailabilityLabels_TC015_TC016(t *testing.T) {
	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "available", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: "discontinued", Category: "c", Price: "1", Available: false, FavoritedAt: exportWhen},
	})

	of := parseCSV(t, body)
	if len(of) != 3 {
		t.Fatalf("parsed %d records, want 1 header + 2 rows — an unavailable item must not be dropped", len(of))
	}
	if got := of[1][4]; got != "Tersedia" {
		t.Errorf("available row label = %q, want Tersedia", got)
	}
	if got := of[2][4]; got != "Tidak Tersedia" {
		t.Errorf("unavailable row label = %q, want Tidak Tersedia", got)
	}
}

// TestExportCSVNoteEmptyCells_TC017_TC018 — SQL NULL and the empty string both
// render as an empty final cell, with no placeholder.
func TestExportCSVNoteEmptyCells_TC017_TC018(t *testing.T) {
	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen, Note: nullNote("")},
	})

	of := parseCSV(t, body)
	for _, row := range of[1:] {
		if len(row) != 7 {
			t.Fatalf("row has %d fields, want 7: %q", len(row), row)
		}
		if got := row[6]; got != "" {
			t.Errorf("note = %q, want an empty cell", got)
		}
	}

	if !strings.Contains(body, "2026-04-09,\r\n") {
		t.Error("an empty note must be followed directly by the record terminator")
	}
}

// TestExportCSVUtf8PassThrough_TC019_TC024 — accented Latin, emoji and CJK
// survive byte-for-byte; nothing is normalized, transliterated or escaped.
func TestExportCSVUtf8PassThrough_TC019_TC024(t *testing.T) {
	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "Café", Category: "Bücher", Price: "1", Available: true, FavoritedAt: exportWhen, Note: nullNote("日本語")},
		{ItemID: 2, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen, Note: nullNote("Café — 🎁 for mom")},
	})

	of := parseCSV(t, body)
	if of[1][1] != "Café" || of[1][2] != "Bücher" || of[1][6] != "日本語" {
		t.Errorf("row 1 = %q, want Café/Bücher/日本語 unchanged", of[1])
	}
	if of[2][6] != "Café — 🎁 for mom" {
		t.Errorf("note = %q, want the original bytes unchanged", of[2][6])
	}
}

// TestExportCSVQuoting_TC021_TC022_TC023 — RFC 4180 quoting, quote doubling,
// and embedded CR/LF preserved inside a quoted field rather than being treated
// as the end of the record.
func TestExportCSVQuoting_TC021_TC022_TC023(t *testing.T) {
	body := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: "Notebook, A5", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: `He said "hi", twice`, Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 3, Name: "i", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen, Note: nullNote("line1\r\nline2")},
	})

	if !strings.Contains(body, `1,"Notebook, A5",`) {
		t.Error("a name containing a comma was not quoted")
	}
	if !strings.Contains(body, `2,"He said ""hi"", twice",`) {
		t.Error("embedded double quotes were not doubled inside a quoted field")
	}
	if !strings.Contains(body, `3,i,c,1,Tersedia,2026-04-09,"line1`+"\r\n"+`line2"`+"\r\n") {
		t.Error("a note containing CRLF was not quoted and preserved verbatim")
	}

	of := parseCSV(t, body)
	if len(of) != 4 {
		t.Fatalf("parsed %d records, want 4 — an embedded CRLF split the record", len(of))
	}
	if of[1][1] != "Notebook, A5" || of[2][1] != `He said "hi", twice` || of[3][6] != "line1\r\nline2" {
		t.Errorf("unquoting did not recover the original values: %q", of)
	}
}

// TestExportCSVNeutralizesFormulas_TC025_TC026 — name, category and note get a
// leading apostrophe when the value begins with =, +, - or @, and only then.
func TestExportCSVNeutralizesFormulas_TC025_TC026(t *testing.T) {
	dangerous := []string{"=SUM(A1:A9)", "+1+1", "-1+1", "@SUM(1+1)"}
	columns := []string{"name", "category", "note"}

	rows := make([]dao.FavoriteExportItem, 0, len(dangerous)*len(columns))
	for _, column := range columns {
		for _, value := range dangerous {
			row := dao.FavoriteExportItem{
				ItemID:      int64(len(rows) + 1),
				Name:        "Ordinary",
				Category:    "Ordinary",
				Price:       "1",
				Available:   true,
				FavoritedAt: exportWhen,
				Note:        nullNote("ordinary"),
			}
			switch column {
			case "name":
				row.Name = value
			case "category":
				row.Category = value
			case "note":
				row.Note = nullNote(value)
			}
			rows = append(rows, row)
		}
	}

	cellIndex := map[string]int{"name": 1, "category": 2, "note": 6}

	of := parseCSV(t, exportBody(t, rows))
	for i, row := range of[1:] {
		// The rows were built column-major: each block of len(dangerous) rows
		// carries one dangerous value in one column, and "Ordinary" elsewhere.
		column := columns[i/len(dangerous)]
		want := "'" + dangerous[i%len(dangerous)]

		if got := row[cellIndex[column]]; got != want {
			t.Errorf("row %d (%s) = %q, want %q", i+1, column, got, want)
		}
		for _, other := range columns {
			if other == column {
				continue
			}
			wantOther := "Ordinary"
			if other == "note" {
				wantOther = "ordinary"
			}
			if got := row[cellIndex[other]]; got != wantOther {
				t.Errorf("row %d (%s) = %q, want %q untouched", i+1, other, got, wantOther)
			}
		}
	}

	// TC-026: a leading space puts the value out of scope, and a mid-string
	// character is never touched.
	outOfScope := exportBody(t, []dao.FavoriteExportItem{
		{ItemID: 1, Name: " =SUM(A1:A9)", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
		{ItemID: 2, Name: "Total = 5", Category: "c", Price: "1", Available: true, FavoritedAt: exportWhen},
	})
	parsed := parseCSV(t, outOfScope)
	if parsed[1][1] != " =SUM(A1:A9)" {
		t.Errorf("a leading space was neutralized: %q", parsed[1][1])
	}
	if parsed[2][1] != "Total = 5" {
		t.Errorf("a mid-string = was altered: %q", parsed[2][1])
	}
}

// TestExportCSVZeroFavoritesIsHeaderOnly_TC027 — zero rows is a valid file:
// BOM + header + CRLF, never an empty body and never an error.
func TestExportCSVZeroFavoritesIsHeaderOnly_TC027(t *testing.T) {
	body := exportBody(t, nil)

	if body != csvHeader {
		t.Fatalf("body = %q, want exactly the BOM + header + CRLF", body)
	}
}

// TestExportCSVScopesToAuthUser_TC030_TC031 — the only user identifier the
// export can see is the auth context's; two callers get two different queries.
func TestExportCSVScopesToAuthUser_TC030_TC031(t *testing.T) {
	seen := make([]int64, 0, 2)
	fake := &fakeDAO{listExport: func(userID int64) ([]dao.FavoriteExportItem, error) {
		seen = append(seen, userID)
		return nil, nil
	}}
	svc := favorite.NewService(fake)

	for _, userID := range []int64{1, 2} {
		ctx := authContext()
		ctx.AuthAccessTokenModel.AuthenticationServerUserID = userID
		if _, err := svc.ExportCSV(ctx); err != nil {
			t.Fatalf("ExportCSV for user %d: %v", userID, err)
		}
	}

	if len(seen) != 2 || seen[0] != 1 || seen[1] != 2 {
		t.Fatalf("DAO saw users %v, want [1 2] taken from the auth context", seen)
	}
}

// TestExportCSVPassesUnexpectedFailuresThrough — a failed query is a real
// error (the handler renders a 500), not a silently empty file.
func TestExportCSVPassesUnexpectedFailuresThrough(t *testing.T) {
	boom := stderrors.New("query timeout")
	fake := &fakeDAO{listExport: func(int64) ([]dao.FavoriteExportItem, error) {
		return nil, boom
	}}

	if _, err := favorite.NewService(fake).ExportCSV(authContext()); err != boom {
		t.Fatalf("ExportCSV error = %v, want the DAO's own error back", err)
	}
}

func firstBytes(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

// parseCSV is a minimal RFC 4180 reader for the test: BOM-tolerant, CRLF or LF
// records, quoted fields with doubled quotes, and embedded CR/LF inside quotes.
// It exists only so a test can assert decoded cells without importing
// encoding/csv — which is deliberately not what the implementation uses.
func parseCSV(t *testing.T, body string) [][]string {
	t.Helper()

	s := strings.TrimPrefix(body, "\ufeff")

	var records [][]string
	var record []string
	var field strings.Builder
	inQuotes := false

	for i := 0; i < len(s); {
		switch {
		case inQuotes:
			switch s[i] {
			case '"':
				if i+1 < len(s) && s[i+1] == '"' {
					field.WriteByte('"')
					i += 2
					continue
				}
				inQuotes = false
				i++
			default:
				field.WriteByte(s[i])
				i++
			}
		case s[i] == '"':
			inQuotes = true
			i++
		case s[i] == ',':
			record = append(record, field.String())
			field.Reset()
			i++
		case s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n':
			record = append(record, field.String())
			field.Reset()
			records = append(records, record)
			record = nil
			i += 2
		case s[i] == '\n':
			record = append(record, field.String())
			field.Reset()
			records = append(records, record)
			record = nil
			i++
		default:
			field.WriteByte(s[i])
			i++
		}
	}

	if field.Len() > 0 || len(record) > 0 {
		record = append(record, field.String())
		records = append(records, record)
	}

	if inQuotes {
		t.Fatalf("unterminated quoted field in %q", body)
	}

	return records
}
