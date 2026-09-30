package main

import (
	"fmt"
	"log"
	"net/http"

	configs "schoolify_be/configs"
	connections "schoolify_be/connections"
	featAuthor "schoolify_be/features/authors"
	featUser "schoolify_be/features/users"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	initModules()
	router := chi.NewRouter()

	initPlugins(router)
	internalModules(router)
	router.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header.Get("Origin"))

		w.WriteHeader(http.StatusOK)
	})

	router.Get("/health-check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	var port uint16 = configs.GetConfig().Service.Port
	var host string = configs.GetConfig().Service.Host
	log.Printf("Server running on  %s:%d", host, port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), router)
}

func initModules() {
	configs.InitModule(".env")
	connections.DbPostgres()
	connections.ConnectRedis()
}
func internalModules(router *chi.Mux) {
	featAuthor.InitModule(router)
	featUser.InitModule(router)
	// users.InitModule(router)
	// files.InitModule(router)
	// markets.InitModule(router)
	// stores.InitModule(router)
}
func initPlugins(router *chi.Mux) {
	// Middleware CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Xrf-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(middleware.Recoverer)

}
