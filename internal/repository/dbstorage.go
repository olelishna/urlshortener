package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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

func (db *DBStorage) SaveEntries(ctx context.Context, entries []model.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"urls"},
		[]string{"uuid", "short_url", "original_url"},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			return []any{entries[i].UUID, entries[i].ShortURL, entries[i].OriginalURL}, nil
		}),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
