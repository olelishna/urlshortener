package storage_test

import (
	"context"
	"testing"

	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	"github.com/olelishna/urlshortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	t.Run("with nil ps - in memory only", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)
		require.NotNil(t, store)
	})

	t.Run("with ps - loads data", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		expectedData := map[string]string{
			"abc123": "https://example.com",
			"def456": "https://test.com",
		}
		m.EXPECT().LoadData(mock.Anything).Return(expectedData, nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)
		require.NotNil(t, store)
	})

	t.Run("with ps - LoadData error", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		expectedErr := assert.AnError
		m.EXPECT().LoadData(mock.Anything).Return(nil, expectedErr)

		store, err := storage.NewStore(context.Background(), m)
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}

func TestStore_Save(t *testing.T) {
	t.Run("save with persistent storage", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().SaveEntry(mock.Anything, mock.MatchedBy(func(e model.Entry) bool {
			return e.ShortURL == "abc" && e.OriginalURL == "https://example.com"
		})).Return(nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		ctx := context.Background()
		userID := "550e8400-e29b-41d4-a716-446655440000"

		err = store.Save(ctx, "abc", "https://example.com", userID)
		assert.NoError(t, err)
	})

	t.Run("save with invalid userID", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		err = store.Save(context.Background(), "abc", "https://example.com", "not-a-uuid")
		assert.Error(t, err)
	})

	t.Run("save with nil ps - in memory only", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		userID := "550e8400-e29b-41d4-a716-446655440000"
		err = store.Save(context.Background(), "abc", "https://example.com", userID)
		assert.NoError(t, err)
	})
}

func TestStore_Get(t *testing.T) {
	t.Run("get from in-memory with ps", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(map[string]string{
			"abc": "https://example.com",
		}, nil)
		m.EXPECT().GetLongURL(mock.Anything, "abc").Return("https://example.com", nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		longURL, exists, err := store.Get(context.Background(), "abc")
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("get from in-memory only (no ps)", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		userID := "550e8400-e29b-41d4-a716-446655440000"
		err = store.Save(context.Background(), "abc", "https://example.com", userID)
		require.NoError(t, err)

		longURL, exists, err := store.Get(context.Background(), "abc")
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("get not found", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		_, exists, err := store.Get(context.Background(), "nonexistent")
		assert.False(t, exists)
		assert.Error(t, err)
	})
}

func TestStore_GetShortByLongURL(t *testing.T) {
	t.Run("with persistent storage", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().GetShortByLongURL(mock.Anything, "https://example.com").Return("abc", nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		short, err := store.GetShortByLongURL(context.Background(), "https://example.com")
		require.NoError(t, err)
		assert.Equal(t, "abc", short)
	})

	t.Run("with nil ps", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		_, err = store.GetShortByLongURL(context.Background(), "https://example.com")
		assert.Error(t, err)
		assert.Equal(t, storage.ErrNoPs, err)
	})
}

func TestStore_GetURLsByUser(t *testing.T) {
	t.Run("with persistent storage", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().GetURLsByUser(mock.Anything, "user-123").Return([]repository.UserLinksListItem{
			{RawShortURL: "abc", OriginalURL: "https://example.com"},
		}, nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		urls, err := store.GetURLsByUser(context.Background(), "user-123")
		require.NoError(t, err)
		require.Len(t, urls, 1)
		assert.Equal(t, "https://example.com", urls[0].OriginalURL)
	})

	t.Run("with nil ps", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		_, err = store.GetURLsByUser(context.Background(), "user-123")
		assert.Error(t, err)
		assert.Equal(t, storage.ErrNoPs, err)
	})
}

func TestStore_DeleteItems(t *testing.T) {
	t.Run("delete with persistent storage", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().DeleteURLs(mock.Anything, mock.Anything).Return(nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		batch := []storage.DeleteBatchItem{
			{UserID: "user-1", ShortURLs: []string{"abc", "def"}},
			{UserID: "user-1", ShortURLs: []string{"abc"}},
		}

		err = store.DeleteItems(context.Background(), batch)
		assert.NoError(t, err)
	})

	t.Run("delete with empty batch", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		err = store.DeleteItems(context.Background(), []storage.DeleteBatchItem{})
		assert.NoError(t, err)
	})

	t.Run("delete with nil ps", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		batch := []storage.DeleteBatchItem{
			{UserID: "user-1", ShortURLs: []string{"abc"}},
		}
		err = store.DeleteItems(context.Background(), batch)
		assert.Error(t, err)
		assert.Equal(t, storage.ErrNoPs, err)
	})

	t.Run("delete multiple users", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().DeleteURLs(mock.Anything, mock.MatchedBy(func(byUser map[string][]string) bool {
			if len(byUser) != 2 {
				return false
			}
			_, ok1 := byUser["user-1"]
			_, ok2 := byUser["user-2"]
			return ok1 && ok2
		})).Return(nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		batch := []storage.DeleteBatchItem{
			{UserID: "user-1", ShortURLs: []string{"abc"}},
			{UserID: "user-2", ShortURLs: []string{"def", "ghi"}},
		}

		err = store.DeleteItems(context.Background(), batch)
		assert.NoError(t, err)
	})
}

func TestStore_SaveBatch(t *testing.T) {
	t.Run("save batch with persistent storage", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)
		m.EXPECT().SaveEntries(mock.Anything, mock.MatchedBy(func(entries []model.Entry) bool {
			return len(entries) == 2
		})).Return(nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		items := []storage.SaveBatchItem{
			{ShortURL: "abc", LongURL: "https://example.com"},
			{ShortURL: "def", LongURL: "https://test.com"},
		}

		userID := "550e8400-e29b-41d4-a716-446655440000"
		err = store.SaveBatch(context.Background(), items, userID)
		assert.NoError(t, err)
	})

	t.Run("save batch with invalid userID", func(t *testing.T) {
		m := repository.NewMockPersistentStorage(t)
		m.EXPECT().LoadData(mock.Anything).Return(make(map[string]string), nil)

		store, err := storage.NewStore(context.Background(), m)
		require.NoError(t, err)

		items := []storage.SaveBatchItem{
			{ShortURL: "abc", LongURL: "https://example.com"},
		}

		err = store.SaveBatch(context.Background(), items, "not-a-uuid")
		assert.Error(t, err)
	})

	t.Run("save batch with nil ps", func(t *testing.T) {
		store, err := storage.NewStore(context.Background(), nil)
		require.NoError(t, err)

		items := []storage.SaveBatchItem{
			{ShortURL: "abc", LongURL: "https://example.com"},
		}

		userID := "550e8400-e29b-41d4-a716-446655440000"
		err = store.SaveBatch(context.Background(), items, userID)
		assert.NoError(t, err)
	})
}
