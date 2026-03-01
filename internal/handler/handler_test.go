package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/olelishna/urlshortener/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ShortenURL(t *testing.T) {
	type fields struct {
		store StoreInterface
	}
	store := storage.NewStore()
	tests := []struct {
		name         string
		method       string
		expectedCode int
		expectedBody string
		fields       fields
		body         string
	}{
		{name: "GET/Not Allowed", method: http.MethodGet, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{name: "PUT/Not Allowed", method: http.MethodPut, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{name: "DELETE/Not Allowed", method: http.MethodDelete, expectedCode: http.StatusMethodNotAllowed, expectedBody: ""},
		{name: "POST/Missing URL", method: http.MethodPost, expectedCode: http.StatusBadRequest, expectedBody: "", body: ""},
		{
			name:         "POST/Ok",
			method:       http.MethodPost,
			expectedCode: http.StatusCreated,
			expectedBody: "http://example.com/_nT7Mj9Z",
			body:         "https://practicum.yandex.ru/",
			fields:       fields{store: store},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				store: tt.fields.store,
			}

			r := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			h.ShortenURL(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")

			if tt.expectedBody != "" {
				assert.Equal(t, len(tt.expectedBody), len(w.Body.String()), "Тело ответа не совпадает с ожидаемым")
			}
		})
	}
}

func TestHandler_RedirectURL(t *testing.T) {
	type fields struct {
		store StoreInterface
	}

	store := storage.NewStore()
	store.Save("shorturl", "http://example.com/")

	tests := []struct {
		name         string
		method       string
		expectedCode int
		hash         string
		fields       fields
	}{
		{name: "POST/Not Allowed", method: http.MethodPost, expectedCode: http.StatusMethodNotAllowed},
		{name: "PUT/Not Allowed", method: http.MethodPut, expectedCode: http.StatusMethodNotAllowed},
		{name: "DELETE/Not Allowed", method: http.MethodDelete, expectedCode: http.StatusMethodNotAllowed},
		{
			name:         "GET/URL not found",
			method:       http.MethodGet,
			expectedCode: http.StatusNotFound,
			hash:         "12345678",
			fields:       fields{store: store},
		},
		{
			name:         "GET/Ok",
			method:       http.MethodGet,
			expectedCode: http.StatusTemporaryRedirect,
			hash:         "shorturl",
			fields:       fields{store: store},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				store: tt.fields.store,
			}
			r := httptest.NewRequest(tt.method, "/"+tt.hash, nil)
			w := httptest.NewRecorder()

			h.RedirectURL(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")

		})
	}
}
