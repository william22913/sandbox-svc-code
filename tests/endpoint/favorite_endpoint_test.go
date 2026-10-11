// Package endpoint pins FEAT-001's route wiring.
//
// Registration is the one part of the feature nothing else exercises: the
// service tests never touch the router, and a live stack would only prove the
// paths a hand-written call happens to use. What can silently go wrong here is
// a method that was never registered, a missing path parameter (which would
// leave the delete handler reading an empty item_id), or an operation that
// reaches the swagger document untagged and gets filed under nothing.
//
// No config import — config's init() calls log.Fatal on a bad environment.
package endpoint

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	"github.com/nexsoft-git/nexcommon/docs/swagger"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"

	"github.com/william22913/sandbox-svc-code/constanta"
	"github.com/william22913/sandbox-svc-code/endpoint/favorite_endpoint"
	"github.com/william22913/sandbox-svc-code/services/favorite"
)

// register builds a real controller and registers the real routes against a
// real mux router. A nil DAO is fine: registration never calls the service.
//
// The two validators are not optional here. WrapService builds each route's
// swagger entry on the spot, and the generator dereferences the tag validator
// while reading DTO fields — register before the controller has one wired and
// this panics at startup rather than at request time. The composition root gets
// this ordering right by wiring validators in Init() and registering routes in
// InitEndpoint() afterwards.
func register() *mux.Router {
	r := mux.NewRouter()
	controller := http_validator.NewHTTPController("fixed-internal-token")
	controller.Router(r)
	controller.TagValidator(tag_validator.NewTagValidator())
	controller.BasicValidator(basic_validator.BasicValidator{})
	favorite_endpoint.NewFavoriteEndpoint(controller, favorite.NewService(nil)).RegisterEndpoint()
	return r
}

// TestRoutesAreRegisteredWithTheirMethods walks the router and records which
// methods answer each of the feature's two path templates.
func TestRoutesAreRegisteredWithTheirMethods(t *testing.T) {
	r := register()

	got := map[string][]string{}
	if err := r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tmpl, err := route.GetPathTemplate()
		if err != nil {
			return nil // a route with no template is not ours
		}
		methods, err := route.GetMethods()
		if err != nil {
			return nil
		}
		got[tmpl] = append(got[tmpl], methods...)
		return nil
	}); err != nil {
		t.Fatalf("walking routes: %v", err)
	}

	collection := got[favorite_endpoint.Path]
	if !contains(collection, http.MethodPost) || !contains(collection, http.MethodGet) {
		t.Errorf("%s answers %v, want POST and GET", favorite_endpoint.Path, collection)
	}
	if contains(collection, http.MethodDelete) {
		t.Errorf("%s answers DELETE, which must live on the item path", favorite_endpoint.Path)
	}

	itemPath := favorite_endpoint.Path + "/{" + favorite.PathItemID + "}"
	item := got[itemPath]
	if !contains(item, http.MethodDelete) {
		t.Errorf("%s answers %v, want DELETE", itemPath, item)
	}
	if len(item) != 1 {
		t.Errorf("%s answers %v, want DELETE alone", itemPath, item)
	}
}

// TestDeleteIsDocumentedTaggedAndSecured — the endpoint layer is also the only
// place a route's swagger metadata is declared. An untagged operation still
// appears in the generated document, just filed under nothing, and an
// operation with no security entry documents itself as publicly callable.
//
// DELETE is the route to assert this on because it is alone on its path, so it
// is the one this feature documents unconditionally (see the limitation test
// below for why the collection path is different).
func TestDeleteIsDocumentedTaggedAndSecured(t *testing.T) {
	register()

	path := favorite_endpoint.Path + "/{" + favorite.PathItemID + "}"
	methods, ok := swagger.SwaggerContainer.SwaggerPath[path]
	if !ok {
		t.Fatalf("swagger documents no route at %s", path)
	}
	op, ok := methods["delete"]
	if !ok {
		t.Fatalf("swagger documents no DELETE %s", path)
	}
	if len(op.Tags) != 1 || op.Tags[0] != constanta.TagFavorite {
		t.Errorf("DELETE %s tags = %v, want [%s]", path, op.Tags, constanta.TagFavorite)
	}
	if len(op.Security) == 0 {
		t.Errorf("DELETE %s documents no security requirement", path)
	}
}

// TestCollectionPathDocumentsEveryMethod — POST and GET both live on
// /v1/favorites, and the generated document must carry both.
//
// This replaces a test that pinned the opposite. Under nexcommon
// v0.3.57-swagger-alpha, NewHandleFuncParam assigned a fresh method map on every
// call (`SwaggerPath[path] = make(...)`), so two routes on one path overwrote
// each other and only the last-registered survived — which is why
// api/openapi.yaml was corrected by hand. The library now guards that assignment
// (`if SwaggerPath[path] == nil`, param.go:21), so the write is additive and
// both operations survive.
//
// If this test ever fails with one method listed, the overwrite regression is
// back and the OpenAPI artifact needs its hand-correction again.
func TestCollectionPathDocumentsEveryMethod(t *testing.T) {
	register()

	methods := swagger.SwaggerContainer.SwaggerPath[favorite_endpoint.Path]
	for _, want := range []string{"post", "get"} {
		if _, ok := methods[want]; !ok {
			t.Fatalf("%s documents %v, want both post and get", favorite_endpoint.Path, methodNames(methods))
		}
	}
}

func methodNames(methods swagger.SwaggerControllerMethods) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	return names
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
