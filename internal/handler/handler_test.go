package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	"github.com/olelishna/urlshortener/internal/compress"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	"github.com/olelishna/urlshortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShortenURL(t *testing.T) {
	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(context.Background()).Return(make(map[string]string), nil)
	m.On("SaveEntry", mock.Anything, mock.AnythingOfType("model.Entry")).
		Return(nil)

	store, _ := storage.NewStore(context.Background(), m)
	h := &handler.Handler{
		Store: store,
	}

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Post("/", h.ShortenURL)

	srv := httptest.NewServer(r)
	defer srv.Close()

	tests := []struct {
		name           string
		method         string
		expectedCode   int
		expectedURLLen int
		body           string
	}{
		{
			name:           "GET/Not Allowed",
			method:         http.MethodGet,
			expectedCode:   http.StatusMethodNotAllowed,
			expectedURLLen: 0,
		},
		{
			name:           "PUT/Not Allowed",
			method:         http.MethodPut,
			expectedCode:   http.StatusMethodNotAllowed,
			expectedURLLen: 0,
		},
		{
			name:           "DELETE/Not Allowed",
			method:         http.MethodDelete,
			expectedCode:   http.StatusMethodNotAllowed,
			expectedURLLen: 0,
		},
		{
			name:           "POST/Missing URL",
			method:         http.MethodPost,
			expectedCode:   http.StatusBadRequest,
			expectedURLLen: 0,
			body:           "",
		},
		{
			name:           "POST/Ok",
			method:         http.MethodPost,
			expectedCode:   http.StatusCreated,
			expectedURLLen: 8,
			body:           "https://practicum.yandex.ru/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL
			req.SetBody(tt.body)

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(
				t,
				tt.expectedCode,
				resp.StatusCode(),
				"Код ответа не совпадает с ожидаемым",
			)

			if tt.expectedURLLen != 0 {
				rawURL := string(resp.Body())
				parsedURL, err := url.Parse(rawURL)
				assert.NoError(t, err, "Error parsing URL")

				assert.Equal(
					t,
					tt.expectedURLLen,
					len(parsedURL.Path),
					"Тело ответа не совпадает с ожидаемым",
				)
			}
		})
	}
}

func TestRedirectURL(t *testing.T) {
	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(context.Background()).Return(make(map[string]string), nil)
	m.On("SaveEntry", mock.Anything, mock.AnythingOfType("model.Entry")).
		Return(nil)

	store, _ := storage.NewStore(context.Background(), m)
	if err := store.Save(
		context.Background(),
		"shorturl",
		"https://practicum.yandex.ru/",
	); err != nil {
		return
	}

	h := &handler.Handler{
		Store: store,
	}

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Get("/{id}", h.RedirectURL)

	srv := httptest.NewServer(r)
	defer srv.Close()

	tests := []struct {
		name         string
		method       string
		expectedCode int
		hash         string
	}{
		{
			name:         "POST/Not Allowed",
			method:       http.MethodPost,
			expectedCode: http.StatusMethodNotAllowed,
			hash:         "shorturl",
		},
		{
			name:         "PUT/Not Allowed",
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
			hash:         "shorturl",
		},
		{
			name:         "DELETE/Not Allowed",
			method:       http.MethodDelete,
			expectedCode: http.StatusMethodNotAllowed,
			hash:         "shorturl",
		},
		{
			name:         "GET/URL not found",
			method:       http.MethodGet,
			expectedCode: http.StatusNotFound,
			hash:         "12345678",
		},
		{name: "GET/Ok", method: http.MethodGet, expectedCode: http.StatusOK, hash: "shorturl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + "/" + tt.hash

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(
				t,
				tt.expectedCode,
				resp.StatusCode(),
				"Код ответа не совпадает с ожидаемым",
			)
		})
	}
}

func TestGzipCompression(t *testing.T) {
	config.ParseFlags()

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(context.Background()).Return(make(map[string]string), nil)
	m.On("SaveEntry", mock.Anything, mock.AnythingOfType("model.Entry")).
		Return(nil)

	store, _ := storage.NewStore(context.Background(), m)

	h := &handler.Handler{
		Store: store,
	}

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Post("/api/shorten", h.ShortenURLJson)

	srv := httptest.NewServer(r)
	defer srv.Close()

	requestBody := `{
		"url": "https://practicum.yandex.ru/"
	}`

	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		req := resty.New().R()
		req.Method = http.MethodPost
		req.URL = srv.URL + "/api/shorten"
		req.SetBody(buf)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "")

		resp, err := req.Send()

		require.NoError(t, err, "error making HTTP request")
		require.Equal(
			t,
			http.StatusCreated,
			resp.StatusCode(),
			"Код ответа не совпадает с ожидаемым",
		)

		body := resp.Body()
		dec := json.NewDecoder(bytes.NewReader(body))

		var shresp model.ShortenResponse

		err = dec.Decode(&shresp)
		require.NoError(t, err)

		validate := validator.New()
		assert.NoError(t, validate.Struct(shresp))
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest(http.MethodPost, srv.URL+"/api/shorten", buf)
		r.RequestURI = ""
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		body, err := io.ReadAll(zr)
		require.NoError(t, err)

		dec := json.NewDecoder(bytes.NewReader(body))

		var shresp model.ShortenResponse

		err = dec.Decode(&shresp)
		require.NoError(t, err)

		validate := validator.New()
		assert.NoError(t, validate.Struct(shresp))
	})
}
