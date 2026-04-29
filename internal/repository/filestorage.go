package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/olelishna/urlshortener/internal/model"
)

type FileStorage struct {
	producer *Producer
	consumer *Consumer
}

func NewFileStorage(filepath string) (*FileStorage, error) {
	producer, err := NewProducer(filepath)
	if err != nil {
		return nil, err
	}

	consumer, err := NewConsumer(filepath)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		producer: producer,
		consumer: consumer,
	}, nil
}

func (f *FileStorage) LoadData(ctx context.Context) (map[string]string, error) {
	defer f.consumer.Close()

	urls := make(map[string]string)

	for {
		entry, err := f.consumer.ReadEntry()
		if err != nil {
			return nil, err
		}

		if entry == nil {
			break
		}

		urls[entry.ShortURL] = entry.OriginalURL
	}

	return urls, nil
}

func (f *FileStorage) SaveEntry(ctx context.Context, entry model.Entry) error {
	if err := f.producer.WriteEntry(&entry); err != nil {
		return err
	}

	return nil
}

func (f *FileStorage) SaveEntries(ctx context.Context, entries []model.Entry) error {
	for _, entry := range entries {
		if err := f.producer.WriteEntry(&entry); err != nil {
			return err
		}
	}

	return nil
}

func (f *FileStorage) GetShortByLongURL(ctx context.Context, longURL string) (string, error) {
	return "", errors.New("not implemented")
}

func (f *FileStorage) GetURLsByUser(ctx context.Context, userID string) ([]UserLinksListItem, error) {
	return nil, errors.New("not implemented")
}

func (f *FileStorage) DeleteURLs(ctx context.Context, urlsByUser map[string][]string) error {
	return errors.New("not implemented")
}

func (f *FileStorage) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	return "", nil
}

type Producer struct {
	file   *os.File
	writer *bufio.Writer
}

func NewProducer(filename string) (*Producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, err
	}

	return &Producer{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (p *Producer) WriteEntry(entry *model.Entry) error {
	data, err := json.Marshal(&entry)
	if err != nil {
		return err
	}

	if _, err := p.writer.Write(data); err != nil {
		return err
	}

	if err := p.writer.WriteByte('\n'); err != nil {
		return err
	}

	return p.writer.Flush()
}

type Consumer struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0o666)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:    file,
		scanner: bufio.NewScanner(file),
	}, nil
}

func (c *Consumer) ReadEntry() (*model.Entry, error) {
	if !c.scanner.Scan() {
		return nil, c.scanner.Err()
	}

	data := c.scanner.Bytes()

	entry := model.Entry{}

	err := json.Unmarshal(data, &entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (c *Consumer) Close() error {
	return c.file.Close()
}
