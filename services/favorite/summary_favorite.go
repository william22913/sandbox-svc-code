package favorite

import (
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"

	svcout "github.com/william22913/sandbox-svc-code/dto/out"
)

// Summary handles GET /favorites/summary — the caller's own totals only.
//
// The user comes from the token, never from the request: this route takes no
// input at all, which is what makes REQ-011 (no request input can influence the
// counts) true by construction rather than by validation.
//
// A failed aggregate is passed through so the formator answers a generic 500.
// Rendering it as 0/0 would be indistinguishable from REQ-012's legitimate
// empty state and would hide an outage.
func (s *Service) Summary(
	ctx *internalCtx.ContextModel,
	_ model.URLParam,
	_ interface{},
) (map[string]string, interface{}, error) {

	userID := ctx.AuthAccessTokenModel.AuthenticationServerUserID

	row, err := s.dao.Summary(userID)
	if err != nil {
		return nil, nil, err
	}

	return nil, svcout.NewFavoriteSummary(row.Total, row.Discontinued), nil
}
