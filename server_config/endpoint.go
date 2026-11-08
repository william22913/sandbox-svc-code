package server_config

import (
	nexendpoint "github.com/nexsoft-git/nexcommon/http/endpoint"

	"github.com/william22913/sandbox-svc-code/endpoint/favorite_endpoint"
)

// InitEndpoint registers this service's routes through nexcommon's endpoint
// aggregator, one AddEndpoint per entity. Must run before GenerateSwaggerDocs
// (called in StartService) — the generator reads metadata that route
// registration accumulates, and called earlier it writes an empty document.
func (s *serverAttribute) InitEndpoint() {
	ep := nexendpoint.NewEndpoint()

	ep.AddEndpoint(favorite_endpoint.NewFavoriteEndpoint(
		s.Validator.HttpController,
		s.Services.Favorite,
	))

	ep.ServeEndpoint()
}
