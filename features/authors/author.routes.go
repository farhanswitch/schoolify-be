package authors

import (
	repoPostgres "schoolify_be/repositories/postgresql"

	"github.com/go-chi/chi/v5"
)

func InitModule(router *chi.Mux) {
	repo := repoPostgres.FactoryAuthorPostgresRepo()
	controller := factoryAuthorController(repo)

	router.Post("/api/pg/v1/author", controller.CreateAuthorPostgresCtrl)
	router.Get("/api/pg/v1/authors", controller.GetAllAuthorPostgresCtrl)
	router.Get("/api/pg/v1/author/{id}", controller.GetAuthorByIDPostgresCtrl)
}
