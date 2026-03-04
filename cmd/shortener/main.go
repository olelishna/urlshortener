package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/storage"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	store := storage.NewStore()
	hand := handler.NewHandler(store)

	r := chi.NewRouter()
	r.Use(middleware.CleanPath, middleware.Recoverer)

	r.Post("/", hand.ShortenURL)
	r.Get("/{id}", hand.RedirectURL)

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
	return nil
}
