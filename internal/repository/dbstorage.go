package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olelishna/urlshortener/internal/model"
)

const (
	_queryTimeOut = 5 * time.Second
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(ctx context.Context, pool *pgxpool.Pool) (*DBStorage, error) {
	ctxT, cancel := context.WithTimeout(ctx, _queryTimeOut)
	defer cancel()

	var exists bool

	err := pool.QueryRow(
		ctxT,
		"SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'urls')",
	).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("table 'urls' not found")
	}

	return &DBStorage{
		pool: pool,
	}, nil
}

func (db *DBStorage) LoadData(ctx context.Context) (map[string]string, error) {
	ctxT, cancel := context.WithTimeout(ctx, _queryTimeOut)
	defer cancel()

	rows, err := db.pool.Query(ctxT, "SELECT short_url, original_url FROM urls")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	urls := make(map[string]string)

	for rows.Next() {
		var entry model.Entry

		err = rows.Scan(&entry.ShortURL, &entry.OriginalURL)
		if err != nil {
			return nil, err
		}

		urls[entry.ShortURL] = entry.OriginalURL
	}

	return urls, nil
}

func (db *DBStorage) SaveEntry(ctx context.Context, entry model.Entry) error {
	_, err := db.pool.Exec(
		ctx,
		"INSERT INTO urls (uuid, short_url, original_url) VALUES ($1,$2,$3)",
		entry.UUID,
		entry.ShortURL,
		entry.OriginalURL,
	)
	if err != nil {
		return err
	}

	return nil
}
