package storage

import (
	"sync"
)

type StoreInterface interface {
	Save(shortURL, longURL string)
	Get(shortURL string) (string, bool)
}

type Store struct {
	urls map[string]string
	mu   sync.RWMutex
}

func NewStore() StoreInterface {
	return &Store{
		urls: make(map[string]string),
	}
}

func (s *Store) Save(shortURL, longURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortURL] = longURL
}

func (s *Store) Get(shortURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	longURL, exists := s.urls[shortURL]

	return longURL, exists
}
