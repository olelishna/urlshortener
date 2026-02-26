package main

import (
	"net/http"

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

	mux := http.NewServeMux()
	mux.HandleFunc(`/`, hand.ShortenURL)
	mux.HandleFunc(`/{id}`, hand.RedirectURL)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
	return nil
}
