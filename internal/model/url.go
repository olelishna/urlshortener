package model

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

type Entry struct {
	UUID        uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
}

func GenerateShortURL() (string, error) {
	const defLen = 8
	randomBytes := make([]byte, defLen)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(randomBytes)[:defLen], nil
}
