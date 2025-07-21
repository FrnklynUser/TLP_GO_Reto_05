package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"acortador-urls/internal/handlers"
	"acortador-urls/internal/shortener"
)

func main() {
	// Crear el servicio de acortador
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := handlers.NewHandler(service)

	// Configurar el router
	r := chi.NewRouter()

	// Middleware básico
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Rutas
	r.Post("/shorten", handler.ShortenURL)
	r.Get("/{short_code}", handler.RedirectURL)

	// Puerto del servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado en puerto %s", port)
	log.Printf("Endpoints disponibles:")
	log.Printf("  POST http://localhost:%s/shorten", port)
	log.Printf("  GET  http://localhost:%s/{short_code}", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Error al iniciar el servidor:", err)
	}
}
