package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olelishna/urlshortener/internal/audit"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	auth "github.com/olelishna/urlshortener/internal/service"
	"github.com/olelishna/urlshortener/internal/storage"
	"go.uber.org/zap"
)

const (
	workerCount             = 3
	chanSize                = 3 // equal to worker, adjustable
	tickerDeletionBatchTime = 5 * time.Second
)

type Handler struct {
	Store    storage.StoreInterface
	deleteCh chan storage.DeleteBatchItem
	notifier *audit.Manager
}

func NewHandler(store storage.StoreInterface, notifier *audit.Manager) *Handler {
	handler := &Handler{
		Store:    store,
		deleteCh: make(chan storage.DeleteBatchItem, chanSize),
		notifier: notifier,
	}

	for w := 1; w <= workerCount; w++ {
		go handler.deleteMessagesWorker(w)
	}

	return handler
}

func (h *Handler) ShortenURL(res http.ResponseWriter, req *http.Request) {
	userID, ok := auth.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		code := http.StatusUnauthorized
		http.Error(res, http.StatusText(code), code)

		return
	}

	longURLRaw, err := io.ReadAll(req.Body)
	if err != nil {
		res.Write([]byte(err.Error()))

		return
	}

	longURL := string(longURLRaw)
	if longURL == "" {
		http.Error(res, "Missing URL", http.StatusBadRequest)

		return
	}

	shortURL, err := model.GenerateShortURL()
	if err != nil {
		logger.Log.Error(err.Error(), zap.String("event", "generate short URL"))

		code := http.StatusInternalServerError
		http.Error(res, http.StatusText(code), code)

		return
	}

	errS := h.Store.Save(req.Context(), shortURL, longURL, userID)
	if errS != nil {
		if errors.Is(errS, repository.ErrNonUnique) {
			oldShortURL, errG := h.Store.GetShortByLongURL(req.Context(), longURL)
			if errG == nil {
				uRes, _ := url.JoinPath(config.FlagBaseURLResult, oldShortURL)

				res.Header().Set("content-type", "text/plain")
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(uRes))

				return
			}

			errS = errors.Join(errS, errG)
		}

		http.Error(res, errS.Error(), http.StatusInternalServerError)

		return
	}

	uRes, _ := url.JoinPath(config.FlagBaseURLResult, shortURL)

	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(uRes))

	h.notifier.Notify(audit.NewEvent("shorten", userID, longURL))
}

func (h *Handler) RedirectURL(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "id")

	longURL, _, err := h.Store.Get(req.Context(), shortURL)
	if err != nil {
		if errors.Is(err, repository.ErrUrlDeleted) {
			code := http.StatusGone
			http.Error(res, http.StatusText(code), code)

			return
		}

		logger.Log.Error(
			err.Error(),
			zap.String("event", "redirect url"),
			zap.String("shortURL", shortURL),
		)

		http.Error(res, "URL not found", http.StatusNotFound)

		return
	}

	http.Redirect(res, req, longURL, http.StatusTemporaryRedirect)

	userID, _ := auth.GetUserIDFromContext(req.Context())

	h.notifier.Notify(audit.NewEvent("follow", userID, longURL))
}

func (h *Handler) ShortenURLJson(res http.ResponseWriter, req *http.Request) {
	userID, ok := auth.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		code := http.StatusUnauthorized
		http.Error(res, http.StatusText(code), code)

		return
	}

	var shreq model.ShortenRequest

	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&shreq); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)

		return
	}

	shortURL, err := model.GenerateShortURL()
	if err != nil {
		logger.Log.Error(err.Error(), zap.String("event", "generate short URL"))

		code := http.StatusInternalServerError
		http.Error(res, http.StatusText(code), code)

		return
	}

	errS := h.Store.Save(req.Context(), shortURL, shreq.URL, userID)
	if errS != nil {
		if errors.Is(errS, repository.ErrNonUnique) {
			oldShortURL, errG := h.Store.GetShortByLongURL(req.Context(), shreq.URL)
			if errG == nil {
				uRes, _ := url.JoinPath(config.FlagBaseURLResult, oldShortURL)
				found := model.ShortenResponse{
					Result: uRes,
				}

				res.Header().Set("Content-Type", "application/json")
				res.WriteHeader(http.StatusConflict)

				enc := json.NewEncoder(res)
				if err := enc.Encode(found); err != nil {
					logger.Log.Debug("error encoding response", zap.Error(err))

					return
				}

				return
			}

			errS = errors.Join(errS, errG)
		}

		http.Error(res, errS.Error(), http.StatusInternalServerError)

		return
	}

	uRes, _ := url.JoinPath(config.FlagBaseURLResult, shortURL)
	result := model.ShortenResponse{
		Result: uRes,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(res)
	if err := enc.Encode(result); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))

		return
	}

	h.notifier.Notify(audit.NewEvent("shorten", userID, shreq.URL))
}

func (h *Handler) ShortenURLBatch(res http.ResponseWriter, req *http.Request) {
	userID, ok := auth.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		code := http.StatusUnauthorized
		http.Error(res, http.StatusText(code), code)

		return
	}

	var shbreq model.ShortenBatchRequest

	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&shbreq); err != nil {
		http.Error(res, "cannot decode request JSON body", http.StatusInternalServerError)

		return
	}

	var (
		shbresp    model.ShortenBatchResponse
		batchItems []storage.SaveBatchItem
	)

	for _, item := range shbreq {
		shortURL, err := model.GenerateShortURL()
		if err != nil {
			logger.Log.Error(err.Error(), zap.String("event", "generate short URL"))

			code := http.StatusInternalServerError
			http.Error(res, http.StatusText(code), code)

			return
		}

		batchItem := storage.SaveBatchItem{
			ShortURL: shortURL,
			LongURL:  item.OriginalURL,
		}
		uRes, _ := url.JoinPath(config.FlagBaseURLResult, shortURL)
		resItem := model.ShortenBatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      uRes,
		}
		shbresp = append(shbresp, resItem)
		batchItems = append(batchItems, batchItem)
	}

	err := h.Store.SaveBatch(req.Context(), batchItems, userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(res)
	if err := enc.Encode(shbresp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))

		return
	}
}

func (h *Handler) GetUserURLs(res http.ResponseWriter, req *http.Request) {
	userID, ok := auth.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		code := http.StatusUnauthorized
		http.Error(res, http.StatusText(code), code)

		return
	}

	urls, err := h.Store.GetURLsByUser(req.Context(), userID)
	if err != nil {
		logger.Log.Debug(err.Error(), zap.String("event", "get URLs by user"))

		code := http.StatusInternalServerError
		http.Error(res, http.StatusText(code), code)

		return
	}

	if len(urls) == 0 {
		code := http.StatusNoContent
		http.Error(res, http.StatusText(code), code)

		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(urls); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))

		return
	}
}

func (h *Handler) DeleteUserURLs(res http.ResponseWriter, req *http.Request) {
	userID, ok := auth.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		code := http.StatusUnauthorized
		http.Error(res, http.StatusText(code), code)

		return
	}

	var urls []string

	if err := json.NewDecoder(req.Body).Decode(&urls); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))

		code := http.StatusBadRequest
		http.Error(res, http.StatusText(code), code)

		return
	}

	if len(urls) == 0 {
		http.Error(res, "Empty list", http.StatusBadRequest)
		return
	}

	select {
	case h.deleteCh <- storage.DeleteBatchItem{UserID: userID, ShortURLs: urls}:
		res.WriteHeader(http.StatusAccepted)
	default:
		logger.Log.Debug("delete channel full, dropping request")

		code := http.StatusServiceUnavailable
		http.Error(res, http.StatusText(code), code)
	}
}

func (h *Handler) deleteMessagesWorker(id int) {

	logger.Log.Info("Starting worker", zap.Int("id", id))

	ticker := time.NewTicker(tickerDeletionBatchTime)

	var batch []storage.DeleteBatchItem

	go func() {
		for {
			select {
			case task := <-h.deleteCh:
				batch = append(batch, task)

				if len(batch) >= 50 {
					err := h.Store.DeleteItems(context.Background(), batch)
					if err != nil {
						logger.Log.Error("cannot delete items", zap.Error(err))
						continue
					}

					batch = nil

					logger.Log.Info("Worker job is done", zap.Int("id", id))
				}
			case <-ticker.C:
				if len(batch) == 0 {
					continue
				}

				err := h.Store.DeleteItems(context.Background(), batch)
				if err != nil {
					logger.Log.Error("cannot delete items", zap.Error(err))
					continue
				}

				batch = nil

				logger.Log.Info("Worker job is done", zap.Int("id", id))
			}
		}
	}()
}

type DbHandler struct {
	Pool *pgxpool.Pool
}

func NewDbHandler(pool *pgxpool.Pool) *DbHandler {
	return &DbHandler{Pool: pool}
}

func (h *DbHandler) PingDB(res http.ResponseWriter, req *http.Request) {
	if h.Pool == nil {
		code := http.StatusServiceUnavailable
		http.Error(res, http.StatusText(code), code)

		return
	}

	err := h.Pool.Ping(req.Context())
	if err != nil {
		logger.Log.Error(err.Error(), zap.String("event", "ping db"))

		code := http.StatusInternalServerError
		http.Error(res, http.StatusText(code), code)

		return
	}

	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("Pong"))
}
