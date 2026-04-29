package storage

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/model"
	"github.com/olelishna/urlshortener/internal/repository"
	auth "github.com/olelishna/urlshortener/internal/service"
)

type StoreInterface interface {
	Save(ctx context.Context, shortURL string, longURL string) error
	SaveBatch(ctx context.Context, items []SaveBatchItem) error
	Get(ctx context.Context, shortURL string) (string, bool, error)
	GetShortByLongURL(ctx context.Context, longURL string) (string, error)
	GetURLsByUser(ctx context.Context) ([]UserLinksListItem, error)
	DeleteItems(ctx context.Context, batch []DeleteBatchItem) error
}

type SaveBatchItem struct {
	ShortURL string
	LongURL  string
}

type UserLinksListItem struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type DeleteBatchItem struct {
	UserID    string
	ShortURLs []string
}

type Store struct {
	urls map[string]string
	ps   repository.PersistentStorage
	mu   sync.RWMutex
}

var (
	ErrNoPs          = errors.New("ps is nil")
	ErrNoCurrentUser = errors.New("no user id found in context")
)

func NewStore(ctx context.Context, ps repository.PersistentStorage) (StoreInterface, error) {
	urls := make(map[string]string)

	if ps != nil {
		var err error

		urls, err = ps.LoadData(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &Store{
		urls: urls,
		ps:   ps,
	}, nil
}

func (s *Store) Save(ctx context.Context, shortURL string, longURL string) error {
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok {
		return ErrNoCurrentUser
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

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
			UserUUID:    userUUID,
		}

		if s.ps != nil {
			if errSave := s.ps.SaveEntry(ctx, entry); errSave != nil {
				chSave <- errSave
			}
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
	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok {
		return ErrNoCurrentUser
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

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
				UserUUID:    userUUID,
			}

			entries = append(entries, entry)
		}

		if s.ps != nil {
			if err := s.ps.SaveEntries(ctx, entries); err != nil {
				chSave <- err
			}
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

func (s *Store) Get(ctx context.Context, shortURL string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	longURL, exists := s.urls[shortURL]

	if !exists {
		return "", false, errors.New("shorturl " + shortURL + " not found")
	}

	if s.ps != nil {
		long, err := s.ps.GetLongURL(ctx, shortURL)
		if err != nil {
			return "", false, err
		}
		if long != "" {
			longURL = long
		}
	}

	return longURL, true, nil
}

func (s *Store) GetShortByLongURL(ctx context.Context, longURL string) (string, error) {
	if s.ps != nil {
		short, err := s.ps.GetShortByLongURL(ctx, longURL)
		if err != nil {
			return "", err
		}

		return short, nil
	}

	return "", ErrNoPs
}

func (s *Store) GetURLsByUser(ctx context.Context) ([]UserLinksListItem, error) {
	if s.ps == nil {
		return nil, ErrNoPs
	}

	userID, ok := ctx.Value(auth.UserIDKey).(string)
	if !ok {
		return nil, ErrNoCurrentUser
	}

	urls, err := s.ps.GetURLsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []UserLinksListItem

	for _, urlItem := range urls {
		uRes, _ := url.JoinPath(config.FlagBaseURLResult, urlItem.RawShortURL)
		result = append(result, UserLinksListItem{
			ShortURL:    uRes,
			OriginalURL: urlItem.OriginalURL,
		})
	}

	return result, nil
}

func (s *Store) DeleteItems(ctx context.Context, batch []DeleteBatchItem) error {
	if s.ps == nil {
		return ErrNoPs
	}

	type key struct {
		userID string
		short  string
	}

	unique := make(map[key]struct{})
	for _, t := range batch {
		for _, short := range t.ShortURLs {
			unique[key{t.UserID, short}] = struct{}{}
		}
	}

	if len(unique) == 0 {
		return nil
	}

	byUser := make(map[string][]string)
	for k := range unique {
		byUser[k.userID] = append(byUser[k.userID], k.short)
	}

	err := s.ps.DeleteURLs(ctx, byUser)
	if err != nil {
		return err
	}

	return nil
}
