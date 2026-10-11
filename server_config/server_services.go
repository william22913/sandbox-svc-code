package server_config

// services groups the project's services. Each is constructed with the DAOs
// built in InitDAO() plus whatever built connection it needs.
type services struct{}

func (s *serverAttribute) InitServices() {
	s.Services = services{}
}
