package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/storage"
)

func main() {
	config.ParseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("Running server on", config.FlagRunAddr)

	store := storage.NewStore()
	hand := handler.NewHandler(store)

	r := chi.NewRouter()
	r.Use(middleware.CleanPath, middleware.Recoverer)

	r.Post("/", hand.ShortenURL)
	r.Get("/{id}", hand.RedirectURL)

	err := http.ListenAndServe(config.FlagRunAddr, r)
	if err != nil {
		return err
	}

	return nil
}
