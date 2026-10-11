package context

import (
	"context"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/token/model"
	"github.com/nexsoft-git/nexlogger"
)

// ToContext previously always built from context.Background(), so a
// value built from it (e.g. the context.Context passed to a token
// validator or a DAO call) was never actually tied to the real
// request's own context — a client disconnecting mid-request could
// not cancel anything downstream, no matter how long a query ran. It
// now derives from baseCtx when WithBaseContext has set one (the HTTP
// layer does this for every real request — see controller/http's
// getContext); context.Background() remains the fallback for a
// ContextModel built outside of handling a real request (a
// background job, internal service-to-service work with no inbound
// request to inherit from, or existing test code that never calls
// WithBaseContext), so this is additive, not a behavior change for
// anything that doesn't opt in.
func (c *ContextModel) ToContext() context.Context {
	base := c.baseCtx
	if base == nil {
		base = context.Background()
	}
	return context.WithValue(
		base,
		constanta.ApplicationContextConstanta,
		c,
	)
}

// WithBaseContext sets the context ToContext() derives from instead
// of context.Background() — pass the real request's r.Context() here
// (once, per request) so cancellation/deadline propagation actually
// works. Returns c for fluent construction.
func (c *ContextModel) WithBaseContext(ctx context.Context) *ContextModel {
	c.baseCtx = ctx
	return c
}

func NewContextModel() *ContextModel {
	ctx := new(ContextModel)
	ctx.ClientAccess.Logger = nexlogger.NewloggerModel()
	return ctx
}

type ContextModel struct {
	AuthAccessTokenModel model.AuthAccessTokenModel
	Limitation           Limitation
	Server               Server
	ClientAccess         ClientAccess
	baseCtx              context.Context
}

type ClientAccess struct {
	IdempotencyKey    string
	Timestamp         time.Time
	ClientAccount     int64
	ClientAccountName string
	Logger            nexlogger.LoggerModel
	Headers           map[string]string
	Path              string
	// RequestLocale is the resolved per-request language preference
	// (e.g. from the Accept-Language header) — see controller/http's
	// getContext, which is the only place this gets set. Deliberately
	// on ClientAccess, not AuthAccessTokenModel.Locale: token
	// validation (which runs AFTER getContext in every wrapper) wholesale
	// replaces AuthAccessTokenModel, which would silently wipe out a
	// header-derived locale set before it. ClientAccess is never
	// touched by token validation, so this survives. error.Formator's
	// ReformatErrorMessage and UnbundledErrorMessages.Error() both
	// check this first, falling back to AuthAccessTokenModel.Locale
	// (unchanged, existing behavior) when it's empty — Part 1 item 14:
	// previously the only locale source was whatever was baked into
	// the token at login time, with no per-request override, so a
	// bilingual deployment answered every client in one language.
	RequestLocale string
}

type Server struct {
	IsSignatureCheck bool
	IsSendSignature  bool
	IsInternal       bool
}

type Limitation struct {
	DBSchema        string
	ParsedDBNode    string
	PermissionHave  string
	UserID          int64
	ServiceUserID   int64
	DataScope       map[string]interface{}
	Other           map[string]interface{}
	AdditionalParse interface{}
}
