package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olelishna/urlshortener/internal/compress"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/repository"
	"github.com/olelishna/urlshortener/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const _shutdownTimeout = 15 * time.Second

func main() {
	config.ParseFlags()

	if err := logger.Init(config.FlagLogLevel); err != nil {
		panic(err)
	}

	if err := run(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Panic(err.Error(), zap.String("event", "run server"))
		}
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
	r.Get("/ping", hand.PingDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

		<-exit
		cancel()
	}()

	httpServer := http.Server{
		Addr:    config.FlagRunAddr,
		Handler: r,
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()

		logger.Log.Info("going to shutdown server")

		tCtx, cancelFn := context.WithTimeout(gCtx, _shutdownTimeout)
		defer cancelFn()

		err := httpServer.Shutdown(tCtx)

		if err != nil {
			logger.Log.Error(err.Error(), zap.String("event", "shutdown server"))
		} else {
			logger.Log.Info("server shutdown was successful")
		}

		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
