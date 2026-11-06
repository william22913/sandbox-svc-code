package favorite

import (
	"strconv"
	"strings"

	internalCtx "github.com/nexsoft-git/nexcommon/context"

	"github.com/william22913/sandbox-svc-code/dao"
)

// exportHeader is REQ-004's header row: exact spelling, lower case, that
// column order.
const exportHeader = "item_id,name,category,price,available,favorited_at,note"

// REQ-007's labels, matching FEAT-001's list view.
const (
	availableLabel   = "Tersedia"
	unavailableLabel = "Tidak Tersedia"
)

// utf8BOM is REQ-009's byte-order mark, written once at the start of the file.
const utf8BOM = "\xef\xbb\xbf"

// exportDateLayout is REQ-006's date-only ISO form.
const exportDateLayout = "2006-01-02"

// recordTerminator is REQ-009's line ending. It is only ever written between
// records; bytes inside a quoted field are preserved as they are.
const recordTerminator = "\r\n"

// ExportCSV renders the calling user's own favorites as the file
// REQ-001–REQ-010 describe.
//
// The user id is read only from the auth context — no parameter, header or
// body value is consulted — so the query cannot be steered at another user's
// rows (REQ-013). The whole file is built in memory: FEAT-001's 500-per-user
// cap already bounds it, so there is no temp file and no cleanup path.
func (s *Service) ExportCSV(ctx *internalCtx.ContextModel) ([]byte, error) {
	userID := ctx.AuthAccessTokenModel.AuthenticationServerUserID

	rows, err := s.dao.ListForExport(userID)
	if err != nil {
		return nil, err
	}

	return buildCSV(rows), nil
}

// buildCSV is the pure renderer: DAO rows in, file bytes out, no clock, no
// database, no request.
//
// encoding/csv is deliberately not used. With UseCRLF it rewrites a lone "\n"
// inside a quoted field as "\r\n" and drops a lone "\r", but REQ-009 requires
// field content to survive byte-for-byte and only the *record terminator* to be
// CRLF (TC-023). The writer below quotes exactly the fields that need it and
// copies every other byte unchanged.
func buildCSV(rows []dao.FavoriteExportItem) []byte {
	var b strings.Builder

	// A row can be at most a few hundred bytes and there are at most 500 of
	// them; this is a hint, not a contract.
	b.Grow(len(utf8BOM) + len(exportHeader) + len(recordTerminator) + len(rows)*128)

	b.WriteString(utf8BOM)
	b.WriteString(exportHeader)
	b.WriteString(recordTerminator)

	for _, row := range rows {
		fields := [...]string{
			strconv.FormatInt(row.ItemID, 10),
			neutralizeFormula(row.Name),
			neutralizeFormula(row.Category),
			row.Price,
			availabilityLabel(row.Available),
			row.FavoritedAt.UTC().Format(exportDateLayout),
			neutralizeFormula(row.Note.String),
		}

		for i, field := range fields {
			if i > 0 {
				b.WriteByte(',')
			}
			writeField(&b, field)
		}
		b.WriteString(recordTerminator)
	}

	return []byte(b.String())
}

// writeField writes one RFC 4180 field: quoted when it holds a delimiter,
// quote or line break, with embedded quotes doubled. Every other byte —
// including UTF-8 sequences and embedded CR/LF — is copied as-is.
func writeField(b *strings.Builder, value string) {
	if !strings.ContainsAny(value, ",\"\r\n") {
		b.WriteString(value)
		return
	}

	b.WriteByte('"')
	for i := 0; i < len(value); i++ {
		if value[i] == '"' {
			b.WriteString(`""`)
			continue
		}
		b.WriteByte(value[i])
	}
	b.WriteByte('"')
}

// neutralizeFormula implements REQ-010: a name, category or note that *begins*
// with =, +, - or @ gets a leading apostrophe, so a spreadsheet opening the
// file treats it as text instead of a formula or link. A leading space puts the
// value out of scope (TC-026), and a value with the character mid-string is
// untouched.
func neutralizeFormula(value string) string {
	if value == "" {
		return value
	}

	switch value[0] {
	case '=', '+', '-', '@':
		return "'" + value
	}

	return value
}

func availabilityLabel(available bool) string {
	if available {
		return availableLabel
	}

	return unavailableLabel
}
