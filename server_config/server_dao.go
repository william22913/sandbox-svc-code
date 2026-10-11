package server_config

import "github.com/william22913/sandbox-svc-code/dao"

// listDAO groups the project's DAOs. Each takes the already-built connection
// from Init(), never dials its own.
type listDAO struct {
	Favorite dao.FavoriteDAO
}

func (s *serverAttribute) InitDAO() {
	s.listDAO = listDAO{
		Favorite: dao.NewFavoriteDAO(s.DBConnection),
	}
}
