package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	"github.com/olelishna/urlshortener/internal/storage"
	"go.uber.org/zap"
)

type Handler struct {
	Store storage.StoreInterface
}

func NewHandler(store storage.StoreInterface) *Handler {
	return &Handler{Store: store}
}

func (h *Handler) ShortenURL(res http.ResponseWriter, req *http.Request) {
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

	shortURL := model.GenerateShortURL()

	errSave := h.Store.Save(req.Context(), shortURL, longURL)
	if errSave != nil {
		if errors.Is(errSave, repository.ErrNonUnique) {
			oldShortURL, errGetByLong := h.Store.GetShortByLongURL(req.Context(), longURL)
			if errGetByLong == nil {
				res.Header().Set("content-type", "text/plain")
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(config.FlagBaseURLResult + "/" + oldShortURL))

				return
			}

			errSave = errors.Join(errSave, errGetByLong)
		}

		http.Error(res, errSave.Error(), http.StatusInternalServerError)

		return
	}

	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(config.FlagBaseURLResult + "/" + shortURL))
}

func (h *Handler) RedirectURL(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "id")

	longURL, exists, err := h.Store.Get(req.Context(), shortURL)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	if !exists {
		http.Error(res, "URL not found", http.StatusNotFound)

		return
	}

	http.Redirect(res, req, longURL, http.StatusTemporaryRedirect)
}

func (h *Handler) ShortenURLJson(res http.ResponseWriter, req *http.Request) {
	logger.Log.Debug("decoding request")

	var shreq model.ShortenRequest

	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&shreq); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)

		return
	}

	shortURL := model.GenerateShortURL()

	errSave := h.Store.Save(req.Context(), shortURL, shreq.URL)
	if errSave != nil {
		if errors.Is(errSave, repository.ErrNonUnique) {
			oldShortURL, errGetByLong := h.Store.GetShortByLongURL(req.Context(), shreq.URL)
			if errGetByLong == nil {

				found := model.ShortenResponse{
					Result: config.FlagBaseURLResult + "/" + oldShortURL,
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

			errSave = errors.Join(errSave, errGetByLong)
		}

		http.Error(res, errSave.Error(), http.StatusInternalServerError)

		return
	}

	result := model.ShortenResponse{
		Result: config.FlagBaseURLResult + "/" + shortURL,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(res)
	if err := enc.Encode(result); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))

		return
	}

	logger.Log.Debug("sending HTTP 201 response")
}

func (h *Handler) PingDB(res http.ResponseWriter, req *http.Request) {
	conn, err := pgx.Connect(req.Context(), config.FlagDatabaseDSN)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	defer conn.Close(req.Context())

	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("Pong"))
}

func (h *Handler) ShortenURLBatch(res http.ResponseWriter, req *http.Request) {
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
		shortURL := model.GenerateShortURL()
		batchItem := storage.SaveBatchItem{
			ShortURL: shortURL,
			LongURL:  item.OriginalURL,
		}
		resItem := model.ShortenBatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      config.FlagBaseURLResult + "/" + shortURL,
		}
		shbresp = append(shbresp, resItem)
		batchItems = append(batchItems, batchItem)
	}

	err := h.Store.SaveBatch(req.Context(), batchItems)
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
