package storage

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
)

type StoreInterface interface {
	Save(ctx context.Context, shortURL string, longURL string) error
	SaveBatch(ctx context.Context, items []SaveBatchItem) error
	Get(ctx context.Context, shortURL string) (string, bool, error)
	GetShortByLongURL(ctx context.Context, longURL string) (string, error)
}

type SaveBatchItem struct {
	ShortURL string
	LongURL  string
}

type Store struct {
	urls              map[string]string
	persistentStorage repository.PersistentStorage
	mu                sync.RWMutex
}

func NewStore(ctx context.Context, persistentStorage repository.PersistentStorage) StoreInterface {
	urls, err := persistentStorage.LoadData(ctx)
	if err != nil {
		panic(err)
	}

	return &Store{
		urls:              urls,
		persistentStorage: persistentStorage,
	}
}

func (s *Store) Save(ctx context.Context, shortURL string, longURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chSave := make(chan error)

	go func() {
		uuidEntry, err := uuid.NewUUID()
		if err != nil {
			chSave <- err
		}

		entry := model.Entry{
			UUID:        uuidEntry,
			ShortURL:    shortURL,
			OriginalURL: longURL,
		}

		if errSave := s.persistentStorage.SaveEntry(ctx, entry); errSave != nil {
			chSave <- errSave
		}

		s.urls[shortURL] = longURL

		chSave <- nil
	}()

	select {
	case err := <-chSave:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Store) SaveBatch(ctx context.Context, items []SaveBatchItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chSave := make(chan error)

	go func() {
		var entries []model.Entry

		for _, item := range items {
			uuidEntry, err := uuid.NewUUID()
			if err != nil {
				chSave <- err
			}

			entry := model.Entry{
				UUID:        uuidEntry,
				ShortURL:    item.ShortURL,
				OriginalURL: item.LongURL,
			}

			entries = append(entries, entry)
		}

		if err := s.persistentStorage.SaveEntries(ctx, entries); err != nil {
			chSave <- err
		}

		for _, item := range items {
			s.urls[item.ShortURL] = item.LongURL
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

func (s *Store) GetShortByLongURL(ctx context.Context, longURL string) (string, error) {
	short, err := s.persistentStorage.GetShortByLongURL(ctx, longURL)
	if err != nil {
		return "", err
	}

	return short, nil
}
