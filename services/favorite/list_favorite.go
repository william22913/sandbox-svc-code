package favorite

import (
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"

	svcout "github.com/william22913/sandbox-svc-code/dto/out"
)

// List handles GET /favorites — the caller's own list, ordered by the DAO.
//
// Unordered by any caller-supplied parameter, as the deferred sort/filter pass
// specifies; the ordering the DAO applies is by recency, which is why
// FavoritedAt is in the payload.
func (s *Service) List(
	ctx *internalCtx.ContextModel,
	_ model.URLParam,
	_ interface{},
) (map[string]string, interface{}, error) {

	userID := ctx.AuthAccessTokenModel.AuthenticationServerUserID

	items, err := s.dao.List(userID)
	if err != nil {
		return nil, nil, err
	}

	// Mapped rather than returned as-is: the DAO's row type carries no JSON
	// contract, and this response must not change shape just because the table
	// gains a column.
	favorites := make([]svcout.FavoriteItem, 0, len(items))
	for _, item := range items {
		favorites = append(favorites, svcout.FavoriteItem{
			ItemID:      item.ItemID,
			Name:        item.Name,
			Available:   item.Available,
			FavoritedAt: item.FavoritedAt,
		})
	}

	return nil, svcout.NewFavoriteList(favorites), nil
}
