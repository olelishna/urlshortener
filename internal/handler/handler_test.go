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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olelishna/urlshortener/internal/audit"
	"github.com/olelishna/urlshortener/internal/compress"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/handler"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	auth "github.com/olelishna/urlshortener/internal/service"
	"github.com/olelishna/urlshortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShortenURL(t *testing.T) {
	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
	m.EXPECT().SaveEntry(mock.Anything, mock.AnythingOfType("model.Entry")).Return(nil)

	store, err := storage.NewStore(context.Background(), m)
	require.NoError(t, err)

	h := handler.NewHandler(store, audit.NewManager())

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
			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))

			uuid1, err := uuid.NewUUID()
			require.NoError(t, err)

			ctx := auth.SetUserIDToContext(req.Context(), uuid1.String())
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(
				t,
				tt.expectedCode,
				rr.Code,
				"Код ответа не совпадает с ожидаемым",
			)

			if tt.expectedURLLen != 0 {
				rawURL := string(rr.Body.Bytes())
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
	uuid1, err := uuid.NewUUID()
	require.NoError(t, err)

	userID := uuid1.String()
	ctx := auth.SetUserIDToContext(t.Context(), userID)

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
	m.EXPECT().SaveEntry(mock.Anything, mock.AnythingOfType("model.Entry")).Return(nil)
	m.EXPECT().GetLongURL(mock.Anything, "12345678").Return("", pgx.ErrNoRows).Maybe()
	m.EXPECT().
		GetLongURL(mock.Anything, "shorturl").
		Return("https://practicum.yandex.ru/", nil).
		Maybe()

	store, err := storage.NewStore(ctx, m)
	require.NoError(t, err)

	if err := store.Save(
		ctx,
		"shorturl",
		"https://practicum.yandex.ru/",
		userID,
	); err != nil {
		return
	}

	h := handler.NewHandler(store, audit.NewManager())

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
		{
			name:         "GET/Ok",
			method:       http.MethodGet,
			expectedCode: http.StatusTemporaryRedirect,
			hash:         "shorturl",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/"+tt.hash, nil)

			ctx := auth.SetUserIDToContext(t.Context(), userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(
				t,
				tt.expectedCode,
				rr.Code,
				"Код ответа не совпадает с ожидаемым",
			)
		})
	}
}

func TestGzipCompression(t *testing.T) {
	config.ParseFlags()

	uuid1, err := uuid.NewUUID()
	require.NoError(t, err)

	userID := uuid1.String()
	ctx := auth.SetUserIDToContext(t.Context(), userID)

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(ctx).Return(make(map[string]string), nil)
	m.EXPECT().SaveEntry(mock.Anything, mock.AnythingOfType("model.Entry")).Return(nil)

	store, err := storage.NewStore(ctx, m)
	require.NoError(t, err)

	h := handler.NewHandler(store, audit.NewManager())

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

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/shorten",
			bytes.NewBufferString(requestBody),
		)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(
			t,
			http.StatusCreated,
			rr.Code,
			"Код ответа не совпадает с ожидаемым",
		)

		body := rr.Body.Bytes()
		dec := json.NewDecoder(bytes.NewReader(body))

		var shresp model.ShortenResponse

		err = dec.Decode(&shresp)
		require.NoError(t, err)

		validate := validator.New()
		assert.NoError(t, validate.Struct(shresp))
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/shorten",
			bytes.NewBufferString(requestBody),
		)

		req.Header.Set("Accept-Encoding", "gzip")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code)

		zr, err := gzip.NewReader(rr.Body)
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

func TestShortenURLBatch(t *testing.T) {
	uuid1, err := uuid.NewUUID()
	require.NoError(t, err)

	userID := uuid1.String()
	ctx := auth.SetUserIDToContext(t.Context(), userID)

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
	m.EXPECT().SaveEntries(mock.Anything, mock.AnythingOfType("[]model.Entry")).Return(nil)

	store, err := storage.NewStore(ctx, m)
	require.NoError(t, err)

	h := handler.NewHandler(store, audit.NewManager())

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Post("/api/shorten/batch", h.ShortenURLBatch)

	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("success batch", func(t *testing.T) {
		body := `[
			{"correlation_id":"1","original_url":"https://example.com"},
			{"correlation_id":"2","original_url":"https://test.com"}
		]`

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/shorten/batch",
			bytes.NewBufferString(body),
		)
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code)

		var resp model.ShortenBatchResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp, 2)
		assert.Equal(t, "1", resp[0].CorrelationID)
		assert.Equal(t, "2", resp[1].CorrelationID)
	})

	t.Run("invalid json", func(t *testing.T) {
		body := `{invalid json}`

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/shorten/batch",
			bytes.NewBufferString(body),
		)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/shorten/batch", nil)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})
}

func TestGetUserURLs(t *testing.T) {
	uuid1, err := uuid.NewUUID()
	require.NoError(t, err)

	userID := uuid1.String()
	ctx := auth.SetUserIDToContext(t.Context(), userID)

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
	m.EXPECT().GetURLsByUser(mock.Anything, userID).Return([]repository.UserLinksListItem{
		{RawShortURL: "abc", OriginalURL: "https://example.com"},
	}, nil)

	store, err := storage.NewStore(ctx, m)
	require.NoError(t, err)

	h := handler.NewHandler(store, audit.NewManager())

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Get("/api/user/urls", h.GetUserURLs)

	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("success with urls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)

		var urls []storage.UserLinksListItem
		err := json.Unmarshal(rr.Body.Bytes(), &urls)
		require.NoError(t, err)
		require.Len(t, urls, 1)
		assert.Equal(t, "https://example.com", urls[0].OriginalURL)
	})

	t.Run("no urls found", func(t *testing.T) {
		m2 := repository.NewMockPersistentStorage(t)
		m2.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m2.EXPECT().
			GetURLsByUser(mock.Anything, userID).
			Return([]repository.UserLinksListItem{}, nil)

		store2, err := storage.NewStore(ctx, m2)
		require.NoError(t, err)

		h2 := handler.NewHandler(store2, audit.NewManager())
		r2 := chi.NewRouter()
		r2.Use(
			middleware.CleanPath,
			middleware.Recoverer,
			logger.MiddlewareLogger,
			compress.MiddlewareGzip,
		)
		r2.Get("/api/user/urls", h2.GetUserURLs)

		srv2 := httptest.NewServer(r2)
		defer srv2.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r2.ServeHTTP(rr, req)

		require.Equal(t, http.StatusNoContent, rr.Code)
	})
}

func TestDeleteUserURLs(t *testing.T) {
	uuid1, err := uuid.NewUUID()
	require.NoError(t, err)

	userID := uuid1.String()
	ctx := auth.SetUserIDToContext(t.Context(), userID)

	m := repository.NewMockPersistentStorage(t)
	m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)

	store, err := storage.NewStore(ctx, m)
	require.NoError(t, err)

	h := handler.NewHandler(store, audit.NewManager())

	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.Recoverer,
		logger.MiddlewareLogger,
		compress.MiddlewareGzip,
	)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("success delete", func(t *testing.T) {
		body := `["abc", "def"]`

		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(body))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusAccepted, rr.Code)
	})

	t.Run("empty list", func(t *testing.T) {
		body := `[]`

		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(body))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		body := `{invalid}`

		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(body))
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})
}

func TestPingDB(t *testing.T) {
	t.Run("no pool", func(t *testing.T) {
		h := handler.NewDbHandler(nil)
		r := chi.NewRouter()
		r.Use(middleware.CleanPath, middleware.Recoverer)
		r.Get("/ping", h.PingDB)

		srv := httptest.NewServer(r)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/ping")
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("ping fails", func(t *testing.T) {
		pool, err := pgxpool.New(context.Background(), "postgres://nonexistent:5432/testdb")
		require.NoError(t, err)
		defer pool.Close()

		time.Sleep(100 * time.Millisecond)

		h := handler.NewDbHandler(pool)
		r := chi.NewRouter()
		r.Use(middleware.CleanPath, middleware.Recoverer)
		r.Get("/ping", h.PingDB)

		srv := httptest.NewServer(r)
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/ping")
		require.NoError(t, err)

		assert.Contains(
			t,
			[]int{http.StatusInternalServerError, http.StatusServiceUnavailable},
			resp.StatusCode,
		)
	})
}
