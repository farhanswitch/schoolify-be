package roles

import (
	repoPostgres "schoolify_be/repositories/postgresql"
	"schoolify_be/utilities"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryRoleRepo()
	controller := factoryRoleController(repo)

	router.With(utilities.AuthMiddleware).Get("/api/role/list", controller.getAllRolesCtrl)
	router.With(utilities.AuthMiddleware).Post("/api/role", controller.createCtrl)
	router.With(utilities.AuthMiddleware).Put("/api/role", controller.updateCtrl)

}
