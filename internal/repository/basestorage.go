package repository

import "github.com/olelishna/urlshortener/internal/model"

type StorageInterface interface {
	LoadData() map[string]string
	SaveEntry(entry model.Entry) error
}
