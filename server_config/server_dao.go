package server_config

// listDAO groups the project's DAOs. Each takes the already-built connection
// from Init(), never dials its own.
type listDAO struct{}

func (s *serverAttribute) InitDAO() {
	s.listDAO = listDAO{}
}
