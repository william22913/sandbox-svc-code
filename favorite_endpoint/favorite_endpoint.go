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
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// Path is the collection these routes live under. The /v1 prefix matches
// nexcommon's own route convention and this service's existing /v1/health —
// architecture.md writes the paths unprefixed, which is shorthand for the
// resource rather than a statement about versioning; see dev-test-results.md.
const Path = "/v1/favorites"

// MenuAddFavorite gates in.AddFavoriteRequest's required-field validation. The
// menu name is the link between this route and that DTO, so the two move
// together.
const MenuAddFavorite = "favorite.create"

// Register wires the three routes onto the shared controller. Called from the
// composition root's InitEndpoint, which runs before GenerateSwaggerDocs.
func Register(httpController *http_validator.HTTPController, service *favorite.Service) {
	registerAdd(httpController, service)
	registerRemove(httpController, service)
	registerList(httpController, service)
}

// POST /v1/favorites — REQ-001, REQ-005, REQ-007.
func registerAdd(httpController *http_validator.HTTPController, service *favorite.Service) {
	httpController.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path,
			httpController.WrapService(
				http_validator.NewWarpServiceParam(
					service,
					service.Add,
					httpController.UserAccessValidator(),
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
func registerRemove(httpController *http_validator.HTTPController, service *favorite.Service) {
	httpController.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path+"/{"+favorite.PathItemID+"}",
			httpController.WrapService(
				http_validator.NewWarpServiceParam(
					service,
					service.Remove,
					httpController.UserAccessValidator(),
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
func registerList(httpController *http_validator.HTTPController, service *favorite.Service) {
	httpController.HandleFunc(
		http_validator.NewHandleFuncParam(
			Path,
			httpController.WrapService(
				http_validator.NewWarpServiceParam(
					service,
					service.List,
					httpController.UserAccessValidator(),
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
