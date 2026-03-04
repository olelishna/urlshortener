package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/olelishna/urlshortener/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ShortenURL(t *testing.T) {

	store := storage.NewStore()
	h := &Handler{
		store: store,
	}

	handler := http.HandlerFunc(h.ShortenURL)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	tests := []struct {
		name           string
		method         string
		expectedCode   int
		expectedUrlLen int
		body           string
	}{
		{name: "GET/Not Allowed", method: http.MethodGet, expectedCode: http.StatusMethodNotAllowed, expectedUrlLen: 0},
		{name: "PUT/Not Allowed", method: http.MethodPut, expectedCode: http.StatusMethodNotAllowed, expectedUrlLen: 0},
		{name: "DELETE/Not Allowed", method: http.MethodDelete, expectedCode: http.StatusMethodNotAllowed, expectedUrlLen: 0},
		{name: "POST/Missing URL", method: http.MethodPost, expectedCode: http.StatusBadRequest, expectedUrlLen: 0, body: ""},
		{name: "POST/Ok", method: http.MethodPost, expectedCode: http.StatusCreated, expectedUrlLen: 9, body: "https://practicum.yandex.ru/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL
			req.SetBody(tt.body)

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(t, tt.expectedCode, resp.StatusCode(), "Код ответа не совпадает с ожидаемым")

			if tt.expectedUrlLen != 0 {
				rawURL := string(resp.Body())
				parsedURL, err := url.Parse(rawURL)
				assert.NoError(t, err, "Error parsing URL")

				assert.Equal(t, tt.expectedUrlLen, len(parsedURL.Path), "Тело ответа не совпадает с ожидаемым")
			}
		})
	}
}

func TestHandler_RedirectURL(t *testing.T) {

	store := storage.NewStore()
	store.Save("shorturl", "https://www.google.com/")

	h := &Handler{
		store: store,
	}

	handler := http.HandlerFunc(h.RedirectURL)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	tests := []struct {
		name         string
		method       string
		expectedCode int
		hash         string
	}{
		{name: "POST/Not Allowed", method: http.MethodPost, expectedCode: http.StatusMethodNotAllowed},
		{name: "PUT/Not Allowed", method: http.MethodPut, expectedCode: http.StatusMethodNotAllowed},
		{name: "DELETE/Not Allowed", method: http.MethodDelete, expectedCode: http.StatusMethodNotAllowed},
		{name: "GET/URL not found", method: http.MethodGet, expectedCode: http.StatusNotFound, hash: "12345678"},
		{name: "GET/Ok", method: http.MethodGet, expectedCode: http.StatusOK, hash: "shorturl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + "/" + tt.hash

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(t, tt.expectedCode, resp.StatusCode(), "Код ответа не совпадает с ожидаемым")

		})
	}
}
