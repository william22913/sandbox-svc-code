package server_config

import "github.com/william22913/sandbox-svc-code/favorite_endpoint"

// InitEndpoint registers this service's routes. Must run before
// GenerateSwaggerDocs (called in StartService) — the generator reads metadata
// that route registration accumulates, and called earlier it writes an empty
// document.
func (s *serverAttribute) InitEndpoint() {
	favorite_endpoint.Register(s.Validator.HttpController, s.Services.Favorite)
}
