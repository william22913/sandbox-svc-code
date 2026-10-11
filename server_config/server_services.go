package server_config

import "github.com/william22913/sandbox-svc-code/services/favorite"

// services groups the project's services. Each is constructed with the DAOs
// built in InitDAO() plus whatever built connection it needs.
type services struct {
	Favorite *favorite.Service
}

func (s *serverAttribute) InitServices() {
	s.Services = services{
		Favorite: favorite.NewService(s.listDAO.Favorite),
	}
}
