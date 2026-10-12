package favorite

import (
	"strconv"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"

	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	svcerror "github.com/william22913/sandbox-svc-code/error"
)

// Remove handles DELETE /favorites/{item_id}.
//
// A path segment that is not a positive integer answers the same 404 as a
// well-formed id that matches no row. That is deliberate: REQ-008 requires this
// endpoint to give one indistinguishable answer for everything that does not
// name one of the caller's own favorites, and a distinct 400 here would be the
// only way it could answer differently.
func (s *Service) Remove(
	ctx *internalCtx.ContextModel,
	urlParam model.URLParam,
	_ interface{},
) (map[string]string, interface{}, error) {

	itemID, err := strconv.ParseInt(urlParam.Path[PathItemID], 10, 64)
	if err != nil || itemID < 1 {
		return nil, nil, svcerror.ErrFavoriteNotFound
	}

	userID := ctx.AuthAccessTokenModel.AuthenticationServerUserID

	// One scoped DELETE by (user_id, item_id) — never "look it up, then check the
	// owner in Go", which is the shape that grows a distinguishable 403.
	removed, err := s.dao.Remove(userID, itemID)
	if err != nil {
		return nil, nil, err
	}
	if !removed {
		// Either it never existed or it belongs to someone else; REQ-008 says the
		// caller must not be able to tell which.
		return nil, nil, svcerror.ErrFavoriteNotFound
	}

	return nil, svcout.RemoveFavoriteResponse{ItemID: itemID}, nil
}
