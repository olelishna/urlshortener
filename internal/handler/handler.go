package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
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

	err = h.Store.Save(req.Context(), shortURL, longURL)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

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

	err := h.Store.Save(req.Context(), shortURL, shreq.URL)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

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
