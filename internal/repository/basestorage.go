package repository

import (
	"context"

	"github.com/olelishna/urlshortener/internal/model"
)

type PersistentStorage interface {
	LoadData(ctx context.Context) (map[string]string, error)
	SaveEntry(ctx context.Context, entry model.Entry) error
}
