// Package favorite is FEAT-001's business logic: idempotent favorite (REQ-005),
// the per-user cap (REQ-007), un-favorite (REQ-003, REQ-008) and the
// list-with-availability read (REQ-002, REQ-004).
//
// It holds no SQL and writes no HTTP: the DAO owns the statements and
// nexcommon's controller owns the response envelope.
package favorite

import (
	"github.com/william22913/sandbox-svc-code/dao"
	"github.com/william22913/sandbox-svc-code/dto/in"
)

// PathItemID is the route's path-parameter name, used both when registering the
// route and when reading the value back out of model.URLParam.Path.
const PathItemID = "item_id"

// MaxFavoritesPerUser is REQ-007's cap: the 501st distinct favorite is refused.
// It lives here rather than in the DAO because it is policy — the DAO takes the
// limit as an argument so only the mechanism lives there and the test harness
// can exercise the boundary with a small number.
const MaxFavoritesPerUser int64 = 500

// Service is the favorite feature's service layer.
type Service struct {
	dao dao.FavoriteDAO
}

// NewService builds the service over the DAO constructed in the composition
// root. It dials nothing itself.
func NewService(favoriteDAO dao.FavoriteDAO) *Service {
	return &Service{dao: favoriteDAO}
}

// GetDTO is the DTO nexcommon decodes and tag-validates the request body into.
// Only POST /favorites carries a body; the other two routes are registered with
// NotReadBody, and nexcommon skips body decoding entirely for GET.
func (s *Service) GetDTO() interface{} {
	return &in.AddFavoriteRequest{}
}

// GetMultipartDTO — this feature has no multipart route.
func (s *Service) GetMultipartDTO() interface{} {
	return nil
}

// GetListScope — this feature has no data-scoped get-list endpoint. The method
// exists only to satisfy nexcommon's services.Services contract.
func (s *Service) GetListScope() []string {
	return nil
}
