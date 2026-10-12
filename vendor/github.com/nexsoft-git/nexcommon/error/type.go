package error

import (
	"strings"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/out"
)

type Formator interface {
	ReformatErrorMessage(
		param errorMessageParam,
	) out.DefaultErrorResponse

	DefaultLanguage(
		lang string,
	) *formator

	Version(
		version string,
	) *formator

	DefaultInternalCode(
		defaultCode string,
	) *formator
}

type ErrorParam struct {
	Param       interface{}
	IsConverted bool
}

type Converter func(...interface{}) map[string]ErrorParam

type UnbundledErrorMessages struct {
	status   int
	code     error
	reason   string
	param    []interface{}
	function Converter
	ctx      *context.ContextModel
}

// ContextModel, Reason and Param are builder methods meant to be
// called on a package-level sentinel var shared by every request that
// can return this error (the documented usage pattern — see
// NewUnBundledErrorMessages). A sentinel is declared once and reused
// forever, so mutating the receiver in place here would mean two
// concurrent requests writing to the SAME struct's fields — one
// request's .Param(...)/.ContextModel(...)/.Reason(...) call could
// overwrite or interleave with another's, surfacing one request's
// field names, locale/request-id context, or reason text in a
// different request's response. Each of these three methods instead
// returns a shallow copy with only its own field changed, leaving the
// shared sentinel itself untouched — safe to call concurrently from
// any number of goroutines on the same package-level var.
func (u *UnbundledErrorMessages) ContextModel(
	ctx *context.ContextModel,
) *UnbundledErrorMessages {
	clone := *u
	clone.ctx = ctx
	return &clone
}

func (e UnbundledErrorMessages) Error() string {
	if _formator == nil {
		return e.code.Error()
	} else {
		var result string
		language := defaultLanguage

		param := make(map[string]interface{})
		if e.function != nil {
			tempParam := e.function(e.param...)

			if e.ctx != nil {
				// RequestLocale (Accept-Language) wins over the
				// token's own Locale when sent — same precedence as
				// formator.go's ReformatErrorMessage; see
				// ClientAccess.RequestLocale's doc comment.
				if e.ctx.ClientAccess.RequestLocale != "" {
					language = e.ctx.ClientAccess.RequestLocale
				} else if e.ctx.AuthAccessTokenModel.Locale != "" {
					language = e.ctx.AuthAccessTokenModel.Locale
				}
			}

			for key := range tempParam {
				param[key] = tempParam[key].Param
				val, _ := tempParam[key].Param.(string)

				if tempParam[key].IsConverted {
					param[key] = _formator.bundles.ReadMessageBundle("common.constanta", strings.ToUpper(val), language, nil)
				}
			}
		}

		result = _formator.bundles.ReadMessageBundle("common.error", e.code.Error(), language, param)

		if e.reason != "" {
			result += " ,Reason : " + e.reason
		}

		return result
	}
}

func (e *UnbundledErrorMessages) Reason(reason string) *UnbundledErrorMessages {
	clone := *e
	clone.reason = reason
	return &clone
}

func (e *UnbundledErrorMessages) Param(param ...interface{}) *UnbundledErrorMessages {
	clone := *e
	clone.param = param
	return &clone
}
