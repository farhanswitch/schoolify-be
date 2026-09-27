package users

import (
	repoPostgres "schoolify_be/repositories/postgresql"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryUserRepo()
	controller := factoryUserController(repo)

	router.Post("/api/auth/login", controller.UserLoginCtrl)
}
