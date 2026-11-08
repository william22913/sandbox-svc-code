// FEAT-005 endpoint tests: the summary route's auth branch, its success body,
// and the shape of the 500 envelope.
//
// UserAccessValidator resolves the raw Authorization value in Redis, so the
// serving tests stand up a minimal RESP2 server in-process that answers the one
// GET nexcommon issues. That keeps the real validator, the real route and the
// real response writer in the path; only the DAO is a fake, because the
// aggregate SQL belongs to the database tests.
package endpoint

import (
	"bufio"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	"github.com/nexsoft-git/nexcommon/docs/swagger"
	nexerrors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"

	"github.com/william22913/sandbox-svc-code/constanta"
	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/endpoint/favorite_endpoint"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// summaryPath is the route under test.
var summaryPath = favorite_endpoint.Path + "/summary"

// summaryRouter wires the real endpoint through a controller whose token
// validator always answers "unidentified and not a JWT", so UserAccessValidator
// rejects without touching Redis.
func summaryRouter(t *testing.T) http.Handler {
	t.Helper()

	controller := http_validator.NewHTTPController("fixed-internal-token")
	controller.TagValidator(tag_validator.NewTagValidator())
	controller.BasicValidator(basic_validator.BasicValidator{})
	controller.TokenValidator(unauthorizedTokenValidator{})
	controller.Formator(testFormator(t))

	r := mux.NewRouter()
	controller.Router(r)
	favorite_endpoint.NewFavoriteEndpoint(controller, favorite.NewService(nil)).RegisterEndpoint()

	return r
}

// testFormator builds the service's error formator from the real bundles, so
// codes are the ones the running service emits.
func testFormator(t *testing.T) nexerrors.Formator {
	t.Helper()

	bundle, err := bundles.NewBundles(filepath.Join("..", "..", "i18n"), "en-US")
	if err != nil {
		t.Fatalf("loading i18n bundles: %v", err)
	}

	return nexerrors.NewErrorFormator(bundle).
		DefaultInternalCode("E-5-SBX-SRV-001").
		DefaultLanguage("en-US").
		Version("1.0.0")
}

// TestSummaryHandler_TC009_TC010_RejectsUnauthenticated — REQ-013: no token and
// an invalid token both get the shared 401 envelope, never a summary body.
func TestSummaryHandler_TC009_TC010_RejectsUnauthenticated(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
	}{
		{name: "anonymous"},
		{name: "invalid token", header: "not-a-real-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, summaryPath, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			summaryRouter(t).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}

			var envelope struct {
				Success bool `json:"success"`
				Payload struct {
					Status  int    `json:"status"`
					Code    string `json:"code"`
					Message string `json:"message"`
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
			if envelope.Payload.Message == "" {
				t.Error("payload.message is empty")
			}

			// The envelope is {success, header, payload}: counts, had any leaked
			// through, would sit inside payload next to status/code/message.
			var raw struct {
				Payload map[string]interface{} `json:"payload"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("decoding raw envelope: %v", err)
			}
			if _, ok := raw.Payload["total"]; ok {
				t.Error("401 body carries a total field")
			}
			if _, ok := raw.Payload["discontinued"]; ok {
				t.Error("401 body carries a discontinued field")
			}
		})
	}
}

// identifiedTokenValidator accepts every token (skips the JWT check) and lets
// UserAccessValidator proceed to the Redis lookup, which the fake below serves.
type identifiedTokenValidator struct{ unauthorizedTokenValidator }

func (identifiedTokenValidator) IsTokenUnidentified(context.Context, string) (bool, error) {
	return false, nil
}

// summaryDAO stubs the DAO surface the service is built against; only Summary
// carries behaviour, because only it is called by the route under test.
type summaryDAO struct {
	summary func(int64) (dao.FavoriteSummary, error)
	userID  int64
}

func (d *summaryDAO) Add(int64, int64, int64) (dao.AddOutcome, error) { return dao.AddInserted, nil }
func (d *summaryDAO) Remove(int64, int64) (bool, error)               { return false, nil }
func (d *summaryDAO) List(int64) ([]dao.FavoriteItem, error)          { return nil, nil }
func (d *summaryDAO) ListForExport(int64) ([]dao.FavoriteExportItem, error) {
	return nil, nil
}
func (d *summaryDAO) ItemExists(int64) (bool, error) { return true, nil }

func (d *summaryDAO) Summary(userID int64) (dao.FavoriteSummary, error) {
	d.userID = userID
	if d.summary == nil {
		return dao.FavoriteSummary{}, nil
	}
	return d.summary(userID)
}

// summaryRouterServing wires the real endpoint to a real token flow: the
// validator accepts the token and the in-process Redis answers the GET with
// tokenPayload(userID). This is what makes the 200/0/500 paths reachable
// without the compose stack.
func summaryRouterServing(t *testing.T, d dao.FavoriteDAO, token string, userID int64) http.Handler {
	t.Helper()

	controller := http_validator.NewHTTPController("fixed-internal-token")
	controller.TagValidator(tag_validator.NewTagValidator())
	controller.BasicValidator(basic_validator.BasicValidator{})
	controller.TokenValidator(identifiedTokenValidator{})
	controller.Redis(fakeRedis(t, token, tokenPayload(userID)))
	controller.Formator(testFormator(t))

	r := mux.NewRouter()
	controller.Router(r)
	favorite_endpoint.NewFavoriteEndpoint(controller, favorite.NewService(d)).RegisterEndpoint()

	return r
}

// tokenPayload is the minimum Redis value nexcommon's setAuthDataToContext
// needs to derive a user id.
func tokenPayload(userID int64) string {
	// "auth" is the wire field nexcommon's RedisAuthAccessTokenModel reads.
	return fmt.Sprintf(`{"auth":"{\"oauth\":{\"user_id\":\"%d\"}}"}`,
		userID)
}

// fakeRedis answers one RESP2 GET per connection: the stored value for `token`,
// a nil bulk (redis.Nil) for anything else. go-redis v7 sends no handshake
// commands when no password/db is configured.
func fakeRedis(t *testing.T, token, payload string) *redis.Client {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening for the fake Redis: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				reader := bufio.NewReader(c)

				// Answer every GET on the connection so go-redis can reuse it.
				for {
					key, err := readRESPGet(reader)
					if err != nil {
						return
					}
					if key == token {
						if _, err := fmt.Fprintf(c, "$%d\r\n%s\r\n", len(payload), payload); err != nil {
							return
						}
						continue
					}
					if _, err := io.WriteString(c, "$-1\r\n"); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	client := redis.NewClient(&redis.Options{
		Addr:       ln.Addr().String(),
		PoolSize:   1,
		MaxRetries: -1,
	})
	// NewClient starts the pool's idle reaper; only Close stops it. Cleanups
	// run LIFO, so this fires before the listener is closed above.
	t.Cleanup(func() { _ = client.Close() })

	return client
}

// readRESPGet parses exactly `*2\r\n$3\r\nGET\r\n$N\r\n<key>\r\n`.
func readRESPGet(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(line) != "*2" {
		return "", fmt.Errorf("unexpected array header %q", line)
	}

	if _, err := r.ReadString('\n'); err != nil { // $3
		return "", err
	}
	cmd := make([]byte, 3)
	if _, err := io.ReadFull(r, cmd); err != nil {
		return "", err
	}
	// go-redis v7 writes the command name in lower case.
	if !strings.EqualFold(string(cmd), "GET") {
		return "", fmt.Errorf("unexpected command %q", cmd)
	}
	if _, err := r.Discard(2); err != nil { // CRLF
		return "", err
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return "", err
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "$")))
	if err != nil {
		return "", fmt.Errorf("unexpected bulk header %q: %w", line, err)
	}

	key := make([]byte, n)
	if _, err := io.ReadFull(r, key); err != nil {
		return "", err
	}
	_, _ = r.Discard(2)

	return string(key), nil
}

// summaryEnvelope is the success body the handler returns.
type summaryEnvelope struct {
	Success bool `json:"success"`
	Payload struct {
		Total        int64 `json:"total"`
		Discontinued int64 `json:"discontinued"`
	} `json:"payload"`
}

func serveSummary(t *testing.T, d dao.FavoriteDAO, token string, userID int64) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, summaryPath, nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	summaryRouterServing(t, d, token, userID).ServeHTTP(rec, req)
	return rec
}

// TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued — the 200 payload comes
// from the token user's aggregate, not from anything in the request.
func TestSummaryHandler_TC001_ReturnsTotalAndDiscontinued(t *testing.T) {
	d := &summaryDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{Total: 3, Discontinued: 1}, nil
	}}

	rec := serveSummary(t, d, "user-1-token", 1)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var env summaryEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	if !env.Success {
		t.Error("success is false")
	}
	if env.Payload.Total != 3 || env.Payload.Discontinued != 1 {
		t.Errorf("payload = %+v, want {3,1}", env.Payload)
	}
	if d.userID != 1 {
		t.Errorf("DAO got user %d, want the token's user 1", d.userID)
	}
}

// TestSummaryHandler_TC011_IsolatedPerUser — REQ-010 / REQ-026 at the handler:
// each token reaches the DAO with its own user id, so the answers stay disjoint.
func TestSummaryHandler_TC011_IsolatedPerUser(t *testing.T) {
	rows := map[int64]dao.FavoriteSummary{
		1: {Total: 2, Discontinued: 0},
		2: {Total: 6, Discontinued: 4},
	}
	d := &summaryDAO{summary: func(userID int64) (dao.FavoriteSummary, error) {
		return rows[userID], nil
	}}

	for userID, want := range rows {
		rec := serveSummary(t, d, fmt.Sprintf("user-%d-token", userID), userID)

		if rec.Code != http.StatusOK {
			t.Fatalf("user %d: status = %d, want 200", userID, rec.Code)
		}
		var env summaryEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("user %d: decoding %q: %v", userID, rec.Body.String(), err)
		}
		if env.Payload.Total != want.Total || env.Payload.Discontinued != want.Discontinued {
			t.Errorf("user %d payload = %+v, want %+v", userID, env.Payload, want)
		}
		if d.userID != userID {
			t.Errorf("DAO got user %d for user %d's token", d.userID, userID)
		}
	}
}

// TestSummaryHandler_TC012_TC013_RequestInputIsIgnored — REQ-011 at the
// handler: a caller-supplied user id, counts, paging and header change nothing.
func TestSummaryHandler_TC012_TC013_RequestInputIsIgnored(t *testing.T) {
	d := &summaryDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{Total: 4, Discontinued: 1}, nil
	}}

	req := httptest.NewRequest(http.MethodGet,
		summaryPath+"?user_id=2&total=999&discontinued=999&limit=1000&offset=0", nil)
	req.Header.Set("Authorization", "user-1-token")
	req.Header.Set("X-User-Id", "2")

	rec := httptest.NewRecorder()
	summaryRouterServing(t, d, "user-1-token", 1).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var env summaryEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	if env.Payload.Total != 4 || env.Payload.Discontinued != 1 {
		t.Errorf("payload = %+v, want the derived {4,1}", env.Payload)
	}
	if d.userID != 1 {
		t.Errorf("DAO got user %d, want the token's user 1", d.userID)
	}
}

// TestSummaryHandler_TC002_ZeroFavoritesIsA200 — REQ-012 at the handler: 200
// with integer zeros, not 404/204/empty.
func TestSummaryHandler_TC002_ZeroFavoritesIsA200(t *testing.T) {
	rec := serveSummary(t, &summaryDAO{}, "user-3-token", 3)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var env summaryEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	if env.Payload.Total != 0 || env.Payload.Discontinued != 0 {
		t.Errorf("payload = %+v, want {0,0}", env.Payload)
	}
}

// TestSummary_TC015_InternalFailureRenders500Envelope — REQ-017: a DAO failure
// surfaces as the standard 500 envelope with no partial counts.
func TestSummary_TC015_InternalFailureRenders500Envelope(t *testing.T) {
	d := &summaryDAO{summary: func(int64) (dao.FavoriteSummary, error) {
		return dao.FavoriteSummary{}, stderrors.New("catalog availability lookup failed")
	}}

	rec := serveSummary(t, d, "user-1-token", 1)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body.String())
	}

	var env struct {
		Success bool `json:"success"`
		Payload struct {
			Status  int    `json:"status"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	if env.Success {
		t.Error("error envelope reports success true")
	}
	if env.Payload.Status != http.StatusInternalServerError {
		t.Errorf("payload.status = %d, want 500", env.Payload.Status)
	}
	if env.Payload.Code == "" || env.Payload.Message == "" {
		t.Errorf("payload = %+v, want a non-empty code and message", env.Payload)
	}

	// Counts would ride inside payload, next to status/code/message.
	var raw struct {
		Payload map[string]interface{} `json:"payload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decoding raw envelope: %v", err)
	}
	if _, ok := raw.Payload["total"]; ok {
		t.Error("500 body carries a total field")
	}
	if _, ok := raw.Payload["discontinued"]; ok {
		t.Error("500 body carries a discontinued field")
	}
}

// TestSummaryIsDocumentedTaggedAndSecured — the generated swagger document must
// carry the route under the favorites tag and behind the auth scheme, or the
// collection generator files it under nothing and it reads as public.
func TestSummaryIsDocumentedTaggedAndSecured(t *testing.T) {
	summaryRouter(t)

	methods, ok := swagger.SwaggerContainer.SwaggerPath[summaryPath]
	if !ok {
		t.Fatalf("swagger documents no route at %s", summaryPath)
	}
	op, ok := methods["get"]
	if !ok {
		t.Fatalf("swagger documents no GET %s", summaryPath)
	}
	if len(op.Tags) != 1 || op.Tags[0] != constanta.TagFavorite {
		t.Errorf("GET %s tags = %v, want [%s]", summaryPath, op.Tags, constanta.TagFavorite)
	}
	if len(op.Security) == 0 {
		t.Errorf("GET %s documents no security requirement", summaryPath)
	}
}
