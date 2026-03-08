package handler

import (
	"io"
	"net/http"

	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/model"
)

type StoreInterface interface {
	Save(shortURL, longURL string)
	Get(shortURL string) (string, bool)
}

type Handler struct {
	store StoreInterface
}

func NewHandler(store StoreInterface) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ShortenURL(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
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

	shortURL := model.GenerateShortURL()
	h.store.Save(shortURL, longURL)

	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(config.FlagBaseUrlResult + shortURL))

}

func (h *Handler) RedirectURL(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	shortURL := req.URL.Path[1:]

	longURL, exists := h.store.Get(shortURL)
	if !exists {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(res, req, longURL, http.StatusTemporaryRedirect)

}
