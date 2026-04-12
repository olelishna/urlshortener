package storage

import (
	"sync"

	"github.com/google/uuid"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
)

type StoreInterface interface {
	Save(shortURL, longURL string) error
	Get(shortURL string) (string, bool)
}

type Store struct {
	urls    map[string]string
	storage repository.StorageInterface
	mu      sync.RWMutex
}

func NewStore(storage repository.StorageInterface) StoreInterface {
	return &Store{
		urls:    storage.LoadData(),
		storage: storage,
	}
}

func (s *Store) Save(shortURL, longURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortURL] = longURL

	uuidEntry, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	entry := model.Entry{
		UUID:        uuidEntry,
		ShortURL:    shortURL,
		OriginalURL: longURL,
	}

	if err := s.storage.SaveEntry(entry); err != nil {
		return err
	}

	return nil
}

func (s *Store) Get(shortURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	longURL, exists := s.urls[shortURL]

	return longURL, exists
}
