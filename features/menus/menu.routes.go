package menus

import (
	repoPostgres "schoolify_be/repositories/postgresql"
	"schoolify_be/utilities"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryMenuRepo()
	controller := factoryMenuController(repo)

	router.With(utilities.AuthMiddleware).Get("/api/menu/list", controller.getAllMenusCtrl)
	router.With(utilities.AuthMiddleware).Post("/api/menu", controller.createCtrl)
}
