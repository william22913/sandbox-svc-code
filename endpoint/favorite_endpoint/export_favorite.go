package favorite_endpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	nexserver "github.com/nexsoft-git/nexcommon/server"
	text "github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"

	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// NewExportHandler returns FEAT-003's raw HTTP handler for
// GET /v1/favorites/export.
//
// It is raw rather than a WrapService route because nexcommon's response
// writer forces application/json (or the wrong file headers), while REQ-002
// requires text/csv and a quoted Content-Disposition. Authentication is not
// reimplemented: it calls the same UserAccessValidator().Func the wrapped
// routes use, so an anonymous or invalid token is the identical 401 envelope
// (REQ-012) and no data is read.
func NewExportHandler(
	httpValidator *http_validator.HTTPController,
	favoriteService *favorite.Service,
	formator errors.Formator,
) http.HandlerFunc {
	validate := httpValidator.UserAccessValidator()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := newRequestContext(r)

		_, headers := httpValidator.Converter(r)
		if err := validate.Func(ctx.ToContext(), headers); err != nil {
			writeError(w, ctx, formator, err)
			return
		}

		body, err := favoriteService.ExportCSV(ctx)
		if err != nil {
			writeError(w, ctx, formator, err)
			return
		}

		// Bare text/csv: no charset parameter, because the BOM is the charset
		// signal the spec chose (REQ-002). The filename is machine-built, so
		// there is no header-injection surface.
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`,
				ExportFileName(ctx.AuthAccessTokenModel.AuthenticationServerUserID, time.Now())))
		// One user's personal data must not sit in an intermediary cache.
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write(body); err != nil {
			log.Error().Err(err).Caller().Msg("error when write favorites export")
		}
	}
}

// ExportFileName builds REQ-002's filename. The conversion to UTC happens here
// and only here, so the date is the export date in UTC even when the host runs
// in another zone (TC-003). Exported so the boundary case is testable without a
// running server or an injected clock.
func ExportFileName(userID int64, at time.Time) string {
	return fmt.Sprintf("favorites-%d-%s.csv", userID, at.UTC().Format("2006-01-02"))
}

// newRequestContext mirrors the parts of nexcommon's own getContext this raw
// route needs: the request's base context (so cancellation propagates), the
// request id the error envelope reports, and the per-request locale. The token
// validator fills the rest in place.
func newRequestContext(r *http.Request) *internalCtx.ContextModel {
	ctx := internalCtx.NewContextModel().WithBaseContext(r.Context())

	ctx.ClientAccess.Logger.RequestID = nexserver.ReadHeader(r, constanta.RequestIDConstanta)
	if ctx.ClientAccess.Logger.RequestID == "" {
		ctx.ClientAccess.Logger.RequestID = text.GetUUID()
	}
	ctx.ClientAccess.RequestLocale = nexserver.ReadHeader(r, constanta.AcceptLanguageHeaderConstanta)

	return ctx
}

// writeError renders err through the same shared Formator the wrapped routes
// use, so the 401/500 bodies are byte-identical in shape. The status comes from
// the formatted payload, never re-derived from the error.
func writeError(
	w http.ResponseWriter,
	ctx *internalCtx.ContextModel,
	formator errors.Formator,
	err error,
) {
	formatted := formator.ReformatErrorMessage(
		*errors.NewErrorMessageParam(err).WithContext(ctx),
	)

	body, marshalErr := json.Marshal(formatted)
	if marshalErr != nil {
		log.Error().Err(marshalErr).Caller().Msg("error when marshal export error response")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(formatted.Payload.Status)
	if _, writeErr := w.Write(body); writeErr != nil {
		log.Error().Err(writeErr).Caller().Msg("error when write export error response")
	}
}
