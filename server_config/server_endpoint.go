package server_config

// InitEndpoint registers this service's routes. Must run before
// GenerateSwaggerDocs (called in StartService) — the generator reads metadata
// that route registration accumulates, and called earlier it writes an empty
// document.
func (s *serverAttribute) InitEndpoint() {
}
