// Package favorite_endpoint registers FEAT-001's three HTTP routes.
//
// Each route is a thin adapter: it names the menu (which is what gates the body
// DTO's validation), the path parameters, and the response DTO, then hands the
// actual work to favorite.Service. Nothing here decides anything about
// favorites.
package favorite_endpoint

import (
	"net/http"

	http_validator "github.com/nexsoft-git/nexcommon/controller/http"

	"github.com/william22913/sandbox-svc-code/constanta"
	svcout "github.com/william22913/sandbox-svc-code/dto/out"
	"github.com/william22913/sandbox-svc-code/router"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// Path is the collection these routes live under. architecture.md writes the
// paths unprefixed, which reads as shorthand for the resource rather than a
// statement about versioning; the /v1 prefix comes from router.V1PrefixPath,
// the one place this service's API version is written down.
const Path = router.V1PrefixPath + "/favorites"

// MenuAddFavorite gates in.AddFavoriteRequest's required-field validation. The
// menu name is the link between this route and that DTO, so the two move
// together.
const MenuAddFavorite = "favorite.create"

// FavoriteEndpoint owns this feature's routes. It is constructed in the
// composition root and registered through nexcommon's endpoint aggregator.
type FavoriteEndpoint struct {
	httpValidator *http_validator.HTTPController
	srv           *favorite.Service
}

func NewFavoriteEndpoint(
	httpValidator *http_validator.HTTPController,
	srv *favorite.Service,
) *FavoriteEndpoint {
	return &FavoriteEndpoint{httpValidator: httpValidator, srv: srv}
}

// RegisterEndpoint satisfies nexcommon's endpoint.HTTPEndpoint, which is what
// lets the composition root collect it alongside every other entity's endpoint.
// Must run before GenerateSwaggerDocs — the generator reads metadata that
// registration accumulates.
func (e *FavoriteEndpoint) RegisterEndpoint() {
	e.registerAdd()
	e.registerRemove()
	e.registerList()
}

// POST /v1/favorites — REQ-001, REQ-005, REQ-007.
func (e *FavoriteEndpoint) registerAdd() {
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path,
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.Add,
					e.httpValidator.UserAccessValidator(),
				).
					Menu(MenuAddFavorite).
					Tags(constanta.TagFavorite).
					OperationID("addFavorite").
					Description("Add an item to the calling user's favorites. Favoriting an item that is already favorited succeeds and changes nothing.").
					DTOOut(http_validator.NewDTOParam(svcout.AddFavoriteResponse{})),
			),
			http.MethodPost,
		),
	)
}

// DELETE /v1/favorites/{item_id} — REQ-003, REQ-008.
func (e *FavoriteEndpoint) registerRemove() {
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path+"/{"+favorite.PathItemID+"}",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.Remove,
					e.httpValidator.UserAccessValidator(),
				).
					// No body, so nothing may be read — without this nexcommon
					// decodes the empty body and answers 400 before the service
					// ever runs.
					NotReadBody().
					PathParams(favorite.PathItemID).
					Tags(constanta.TagFavorite).
					OperationID("removeFavorite").
					Description("Remove an item from the calling user's favorites. Answers 404 both when the item was never favorited and when it belongs to another user.").
					DTOOut(http_validator.NewDTOParam(svcout.RemoveFavoriteResponse{})),
			),
			http.MethodDelete,
		),
	)
}

// GET /v1/favorites — REQ-002, REQ-004.
//
// Plain WrapService, not WrapServiceListData: the list wrapper's contract
// requires page/limit/filter/order/connector query parameters and a
// []model.SearchParam, none of which mean anything while sorting and filtering
// are explicitly deferred. Registering it as a get-list would promise a
// paginated, filterable endpoint this pass does not implement.
func (e *FavoriteEndpoint) registerList() {
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path,
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.List,
					e.httpValidator.UserAccessValidator(),
				).
					Tags(constanta.TagFavorite).
					OperationID("listFavorites").
					Description("List the calling user's favorites, most recently favorited first. Items discontinued in the catalog are still listed, marked unavailable.").
					DTOOut(http_validator.NewDTOParam(svcout.FavoriteList{})),
			),
			http.MethodGet,
		),
	)
}
