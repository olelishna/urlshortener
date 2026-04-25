package repository

import (
	"context"

	"github.com/olelishna/urlshortener/internal/model"
)

type UserLinksListItem struct {
	RawShortURL string
	OriginalURL string
}

type PersistentStorage interface {
	LoadData(ctx context.Context) (map[string]string, error)
	SaveEntry(ctx context.Context, entry model.Entry) error
	SaveEntries(ctx context.Context, entries []model.Entry) error
	GetShortByLongURL(ctx context.Context, longURL string) (string, error)
	GetURLsByUser(ctx context.Context, userID string) ([]UserLinksListItem, error)
}
