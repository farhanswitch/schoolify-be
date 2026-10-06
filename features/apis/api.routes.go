package apis

import (
	repoPostgres "schoolify_be/repositories/postgresql"
	"schoolify_be/utilities"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryApiRepo()
	controller := factoryApiController(repo)

	router.With(utilities.AuthMiddleware).Get("/api/endpoint/list", controller.getAllCtrl)
	router.With(utilities.AuthMiddleware).Post("/api/endpoint", controller.createCtrl)
	router.With(utilities.AuthMiddleware).Put("/api/endpoint/{apiID}", controller.updateCtrl)
	router.With(utilities.AuthMiddleware).Delete("/api/endpoint/{apiID}", controller.deleteCtrl)

}
