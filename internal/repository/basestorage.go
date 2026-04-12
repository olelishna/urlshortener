package repository

import (
	"context"

	"github.com/olelishna/urlshortener/internal/model"
)

type PersistentStorage interface {
	LoadData(ctx context.Context) (map[string]string, error)
	SaveEntry(ctx context.Context, entry model.Entry) error
}
type BaseStorage struct{}

func NewBaseStorage() *BaseStorage {
	return &BaseStorage{}
}

func (b *BaseStorage) LoadData(ctx context.Context) (map[string]string, error) {
	return make(map[string]string), nil
}

func (b *BaseStorage) SaveEntry(ctx context.Context, entry model.Entry) error {
	return nil
}
