package model

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateShortURL() string {
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)

	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(randomBytes)[:8]
}
