package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olelishna/urlshortener/internal/compress"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/repository"
	"github.com/olelishna/urlshortener/internal/storage"
	"go.uber.org/zap"
)

func main() {
	config.ParseFlags()

	if err := logger.Init(config.FlagLogLevel); err != nil {
		panic(err)
	}

	if err := run(); err != nil {
		logger.Log.Panic(err.Error(), zap.String("event", "run server"))
	}
}

func run() error {
	logger.Log.Info("Running server", zap.String("addr", config.FlagRunAddr))

	fileStorage := repository.NewFileStorage()
	store := storage.NewStore(fileStorage)
	hand := handler.NewHandler(store)

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)

	r.Post("/", hand.ShortenURL)
	r.Get("/{id}", hand.RedirectURL)
	r.Post("/api/shorten", hand.ShortenURLJson)

	if err := http.ListenAndServe(config.FlagRunAddr, r); err != nil {
		return err
	}

	return nil
}
