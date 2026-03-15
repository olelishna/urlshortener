package model

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateShortURL() string {
	const defLen = 8
	randomBytes := make([]byte, defLen)

	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(randomBytes)[:defLen]
}
