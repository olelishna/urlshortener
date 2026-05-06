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
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olelishna/urlshortener/internal/compress"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/repository"
	auth "github.com/olelishna/urlshortener/internal/service"
	"github.com/olelishna/urlshortener/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const ShutdownTimeout = 15 * time.Second

func main() {
	ctx := context.Background()

	config.ParseFlags()
	config.GetEnvParams()

	if err := logger.Init(config.FlagLogLevel); err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "logger initialization"))
	}

	if err := auth.Init(); err != nil {
		logger.Log.Fatal(err.Error(), zap.String("event", "init auth"))
	}

	if err := run(ctx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal(err.Error(), zap.String("event", "run server"))
		}
	}
}

func run(ctx context.Context) error {
	logger.Log.Info("Running server at", zap.String("addr", config.FlagRunAddr))

	var (
		pl *pgxpool.Pool
		ps repository.PersistentStorage = nil
	)

	if config.FlagDatabaseDSN != "" {
		pool, err := pgxpool.New(ctx, config.FlagDatabaseDSN)
		if err != nil {
			return err
		}
		defer pool.Close()

		dbps, err := repository.NewDBStorage(ctx, pool)
		if err != nil {
			return err
		}

		logger.Log.Info("database storage initialized")

		pl = pool
		ps = dbps
	} else if config.FlagFileStoragePath != "" {
		fps, err := repository.NewFileStorage(config.FlagFileStoragePath)
		if err != nil {
			return err
		}

		ps = fps

		logger.Log.Info("file storage initialized")
	} else {
		logger.Log.Info("memory storage initialized")
	}

	store, err := storage.NewStore(ctx, ps)
	if err != nil {
		return err
	}

	hand := handler.NewHandler(store)
	dbHand := handler.NewDbHandler(pl)

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		auth.MiddlewareCheckAuth,
		compress.MiddlewareGzip,
	)

	r.Post("/", hand.ShortenURL)
	r.Get("/{id}", hand.RedirectURL)
	r.Post("/api/shorten", hand.ShortenURLJson)
	r.Post("/api/shorten/batch", hand.ShortenURLBatch)
	r.Get("/api/user/urls", hand.GetUserURLs)
	r.Delete("/api/user/urls", hand.DeleteUserURLs)
	r.Get("/ping", dbHand.PingDB)

	nCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, os.Kill)
	defer stop()

	httpServer := http.Server{
		Addr:    config.FlagRunAddr,
		Handler: r,
	}

	g, gCtx := errgroup.WithContext(nCtx)
	g.Go(func() error {
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()

		logger.Log.Info(gCtx.Err().Error())
		stop()

		logger.Log.Info("going to shutdown server")

		tCtx, cancelFn := context.WithTimeout(gCtx, ShutdownTimeout)
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
