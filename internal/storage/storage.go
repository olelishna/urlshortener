package storage

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
)

type StoreInterface interface {
	Save(ctx context.Context, shortURL, longURL string) error
	Get(ctx context.Context, shortURL string) (string, bool, error)
}

type Store struct {
	urls              map[string]string
	persistentStorage repository.PersistentStorage
	mu                sync.RWMutex
}

func NewStore(persistentStorage repository.PersistentStorage) StoreInterface {
	return &Store{
		urls:              persistentStorage.LoadData(),
		persistentStorage: persistentStorage,
	}
}

func (s *Store) Save(ctx context.Context, shortURL, longURL string) error {
	chSave := make(chan error)

	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.urls[shortURL] = longURL

		uuidEntry, err := uuid.NewUUID()
		if err != nil {
			chSave <- err
		}

		entry := model.Entry{
			UUID:        uuidEntry,
			ShortURL:    shortURL,
			OriginalURL: longURL,
		}

		if err := s.persistentStorage.SaveEntry(entry); err != nil {
			chSave <- err
		}

		chSave <- nil
	}()

	select {
	case err := <-chSave:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type GetAnswer struct {
	LongURL string
	Exist   bool
}

func (s *Store) Get(ctx context.Context, shortURL string) (string, bool, error) {
	chGet := make(chan GetAnswer)

	go func() {
		s.mu.RLock()
		defer s.mu.RUnlock()
		longURL, exists := s.urls[shortURL]

		chGet <- GetAnswer{
			LongURL: longURL,
			Exist:   exists,
		}
	}()

	select {
	case answer := <-chGet:
		return answer.LongURL, answer.Exist, nil
	case <-ctx.Done():
		return "", false, ctx.Err()
	}
}
