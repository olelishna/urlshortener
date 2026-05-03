package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"github.com/olelishna/urlshortener/internal/config"
	"github.com/olelishna/urlshortener/internal/logger"
	"github.com/olelishna/urlshortener/internal/model"
)

const (
	QueryTimeOut = 5 * time.Second
)

type DBStorage struct {
	pool *pgxpool.Pool
}

var (
	ErrNonUnique   = errors.New("data conflict")
	ErrEmptyString = errors.New("no empty string allowed")
	ErrNoUser      = errors.New("user uuid shouldn't be empty")
	ErrUrlDeleted  = errors.New("url is deleted")
)

func NewDBStorage(ctx context.Context, pool *pgxpool.Pool) (*DBStorage, error) {
	ctxT, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	err := applyMigrations()
	if err != nil {
		return nil, err
	}

	var exists bool

	err = pool.QueryRow(
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

func applyMigrations() error {
	logger.Log.Info("start migrations")

	m, err := migrate.New("file://migrations", config.FlagDatabaseDSN)
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}

		logger.Log.Info("database is already up-to-date")
	}

	logger.Log.Info("end migrations")

	return nil
}

func (db *DBStorage) LoadData(ctx context.Context) (map[string]string, error) {
	ctxT, cancel := context.WithTimeout(ctx, QueryTimeOut)
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
		"INSERT INTO urls (uuid, short_url, original_url, user_uuid) VALUES ($1,$2,$3,$4)",
		entry.UUID,
		entry.ShortURL,
		entry.OriginalURL,
		entry.UserUUID,
	)
	if err == nil {
		return nil
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](
		err,
	); ok &&
		pgErr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%w: original_url %s already exists", ErrNonUnique, entry.OriginalURL)
	}

	return err
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
		[]string{"uuid", "short_url", "original_url", "user_uuid"},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			return []any{entries[i].UUID, entries[i].ShortURL, entries[i].OriginalURL, entries[i].UserUUID}, nil
		}),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (db *DBStorage) GetShortByLongURL(ctx context.Context, longURL string) (string, error) {
	if longURL == "" {
		return "", ErrEmptyString
	}

	var shortURL string

	row := db.pool.QueryRow(ctx, "SELECT short_url FROM urls where original_url = $1", longURL)
	if err := row.Scan(&shortURL); err != nil {
		return "", err
	}

	return shortURL, nil
}

func (db *DBStorage) GetURLsByUser(ctx context.Context, userID string) ([]UserLinksListItem, error) {
	if userID == "" {
		return nil, ErrNoUser
	}

	var urls []UserLinksListItem

	ctxT, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	rows, err := db.pool.Query(ctxT, "SELECT short_url, original_url FROM urls where user_uuid = $1 and is_deleted = $2", userID, false)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var item UserLinksListItem

		err = rows.Scan(&item.RawShortURL, &item.OriginalURL)
		if err != nil {
			return nil, err
		}

		urls = append(urls, item)
	}

	return urls, nil
}

func (db *DBStorage) DeleteURLs(ctx context.Context, urlsByUser map[string][]string) error {
	for userID, shorts := range urlsByUser {
		query := `UPDATE urls SET is_deleted = true 
                  WHERE user_uuid = $1 AND short_url = ANY($2)`
		_, err := db.pool.Exec(ctx, query, userID, pq.Array(shorts))
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBStorage) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", ErrEmptyString
	}

	var longURL string
	var isDeleted bool

	row := db.pool.QueryRow(ctx, "SELECT original_url, is_deleted FROM urls where short_url = $1", shortURL)
	if err := row.Scan(&longURL, &isDeleted); err != nil {
		return "", err
	}

	if isDeleted {
		return "", ErrUrlDeleted
	}

	return longURL, nil
}
