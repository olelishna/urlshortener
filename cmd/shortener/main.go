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
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ctx := context.Background()

	config.ParseFlags()

	if err := logger.Init(config.FlagLogLevel); err != nil {
		panic(err)
	}

	if err := run(ctx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Panic(err.Error(), zap.String("event", "run server"))
		}
	}
}

func run(ctx context.Context) error {
	logger.Log.Info("Running server at", zap.String("addr", config.FlagRunAddr))

	var (
		persistentStorage repository.PersistentStorage = repository.NewBaseStorage()
		storageInitErr    error
	)

	logger.Log.Info("memory storage initialized by default")

	if config.FlagDatabaseDSN != "" {
		m, err := migrate.New("file://migrations", config.FlagDatabaseDSN)
		if err != nil {
			return err
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}

		pool, err := pgxpool.New(ctx, config.FlagDatabaseDSN)
		if err != nil {
			panic(err)
		}
		defer pool.Close()

		persistentStorage, storageInitErr = repository.NewDBStorage(ctx, pool)
		if storageInitErr != nil {
			logger.Log.Warn(storageInitErr.Error(), zap.String("event", "init database storage"))
		} else {
			logger.Log.Info("database storage initialized")
		}
	}

	if storageInitErr != nil || config.FlagDatabaseDSN == "" {
		if config.FlagFileStoragePath != "" {
			logger.Log.Info("file storage initialized")

			persistentStorage = repository.NewFileStorage(config.FlagFileStoragePath)
		} else {
			persistentStorage = repository.NewBaseStorage()
		}
	}

	store := storage.NewStore(ctx, persistentStorage)
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

	ctxC, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		<-exit
		cancel()
	}()

	httpServer := http.Server{
		Addr:    config.FlagRunAddr,
		Handler: r,
	}

	g, gCtx := errgroup.WithContext(ctxC)
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
