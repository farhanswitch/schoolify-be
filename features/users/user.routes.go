package users

import (
	repoPostgres "schoolify_be/repositories/postgresql"
	"schoolify_be/utilities"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryUserRepo()
	controller := factoryUserController(repo)

	router.Post("/api/auth/login", controller.UserLoginCtrl)
	router.Post("/api/auth/register", controller.RegisterUserCtrl)
	router.With(utilities.LogoutAuthMiddleware).Post("/api/auth/logout", controller.LogoutCtrl)
	router.With(utilities.AuthMiddleware).Get("/api/user/access-menu", controller.GetAccessMenusCtrl)
}
