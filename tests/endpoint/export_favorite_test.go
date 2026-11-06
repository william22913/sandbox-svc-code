// FEAT-003 endpoint tests: the raw CSV route's authentication branch and the
// filename's UTC boundary.
//
// The 200 branch needs a Redis-backed token (UserAccessValidator resolves the
// raw Authorization value there), so its header set and body are verified
// against the live stack by the Postman collection instead. What is asserted
// here is everything that must hold before any data is read: an anonymous or
// invalid token produces the standard error envelope, never a CSV response.
package endpoint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	tokenmodel "github.com/nexsoft-git/nexcommon/token/model"

	"github.com/william22913/sandbox-svc-code/endpoint/favorite_endpoint"
	"github.com/william22913/sandbox-svc-code/router"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// unauthorizedTokenValidator behaves like a token that is not in Redis and is
// not a valid JWT: getTokenOnRedis then returns ErrUnauthorized without ever
// reaching Redis itself.
type unauthorizedTokenValidator struct{}

func (unauthorizedTokenValidator) ValidateJWTToken(context.Context, string, string, bool) error {
	return errors.ErrUnauthorized
}

func (unauthorizedTokenValidator) IsTokenUnidentified(context.Context, string) (bool, error) {
	return true, nil
}

func (unauthorizedTokenValidator) RefreshToken(context.Context, string, string) (string, string, error) {
	return "", "", nil
}

func (unauthorizedTokenValidator) LogoutToken(context.Context, string) error { return nil }

func (unauthorizedTokenValidator) ValidateTokenWithoutCheckSignature(string, string, string) (tokenmodel.PayloadJWTToken, error) {
	return tokenmodel.PayloadJWTToken{}, nil
}

func (unauthorizedTokenValidator) SetAdditionalRedisValueParser(interface{}) {}

func (unauthorizedTokenValidator) GetAdditionalRedisValueParser() interface{} { return nil }

// exportRouter wires the real handler around the real mux path.
func exportRouter(t *testing.T) http.Handler {
	t.Helper()

	controller := http_validator.NewHTTPController("fixed-internal-token")
	controller.TokenValidator(unauthorizedTokenValidator{})

	bundle, err := bundles.NewBundles(filepath.Join("..", "..", "i18n"), "en-US")
	if err != nil {
		t.Fatalf("loading i18n bundles: %v", err)
	}
	formator := errors.NewErrorFormator(bundle).
		DefaultInternalCode("E-5-SBX-SRV-001").
		DefaultLanguage("en-US").
		Version("1.0.0")

	r := mux.NewRouter()
	r.HandleFunc(
		router.FavoriteExportPath,
		favorite_endpoint.NewExportHandler(controller, favorite.NewService(nil), formator),
	).Methods(http.MethodGet)

	return r
}

// TestExportHandlerRejectsUnauthenticated_TC028_TC029 — no token, and an
// invalid token, both produce the shared 401 envelope and no file: no CSV
// content type and no attachment disposition.
func TestExportHandlerRejectsUnauthenticated_TC028_TC029(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
	}{
		{name: "anonymous"},
		{name: "invalid token", header: "not-a-real-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, router.FavoriteExportPath+"?user_id=2", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			exportRouter(t).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if got := rec.Header().Get("Content-Disposition"); got != "" {
				t.Errorf("Content-Disposition = %q, want none", got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "" {
				t.Errorf("Cache-Control = %q, want none on an error", got)
			}
			if body := rec.Body.String(); len(body) > 0 && body[0] == 0xEF {
				t.Error("error response starts with a UTF-8 BOM, i.e. it is the CSV file")
			}

			var envelope struct {
				Success bool `json:"success"`
				Payload struct {
					Status int    `json:"status"`
					Code   string `json:"code"`
				} `json:"payload"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decoding error envelope %q: %v", rec.Body.String(), err)
			}
			if envelope.Success {
				t.Error("error envelope reports success true")
			}
			if envelope.Payload.Status != http.StatusUnauthorized {
				t.Errorf("payload.status = %d, want 401", envelope.Payload.Status)
			}
			if envelope.Payload.Code != "E-1-CMD-AUT-001" {
				t.Errorf("payload.code = %q, want E-1-CMD-AUT-001", envelope.Payload.Code)
			}
		})
	}
}

// TestExportFileNameUsesUTCDate_TC002_TC003 — the filename carries the user id
// from the auth context and the export date in UTC. The second and third cases
// are the same instants as 23:59:59Z and 00:00:01Z expressed in Jakarta time,
// so a server-local Format would emit the wrong day for both.
func TestExportFileNameUsesUTCDate_TC002_TC003(t *testing.T) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)

	for _, tc := range []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "satisfies REQ-002's shape",
			at:   time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC),
			want: "favorites-7-2026-11-05.csv",
		},
		{
			name: "just before UTC midnight, server at UTC+7",
			at:   time.Date(2026, 11, 6, 6, 59, 59, 0, jakarta), // 2026-11-05T23:59:59Z
			want: "favorites-7-2026-11-05.csv",
		},
		{
			name: "just after UTC midnight, server at UTC+7",
			at:   time.Date(2026, 11, 6, 7, 0, 1, 0, jakarta), // 2026-11-06T00:00:01Z
			want: "favorites-7-2026-11-06.csv",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := favorite_endpoint.ExportFileName(7, tc.at); got != tc.want {
				t.Fatalf("ExportFileName = %q, want %q", got, tc.want)
			}
		})
	}
}
