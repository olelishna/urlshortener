package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/olelishna/urlshortener/internal/config"
)

const UserIDKey string = "userUUID"

const (
	authCookie       = "auth_cookie"
	authCookieMaxAge = 3600 * 24 * 30
)

var aesBlock cipher.Block

func Init() error {
	var err error
	var key []byte

	key, err = hex.DecodeString(config.AesKey)
	if err != nil {
		return err
	}

	aesBlock, err = aes.NewCipher(key)
	if err != nil {
		return err
	}

	return nil
}

func MiddlewareCheckAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ow := w

		id := getUserID(ow, r)
		if err := uuid.Validate(id); err != nil {
			code := http.StatusUnauthorized
			http.Error(ow, http.StatusText(code), code)

			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, id)

		next.ServeHTTP(ow, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

func getUserID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(authCookie)
	if err != nil || cookie == nil {
		id := generateID(w)

		return id
	}

	decrypted, _ := decrypt(cookie.Value)

	return decrypted
}

func generateID(w http.ResponseWriter) string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}

	id := newUUID.String()

	encrypted, _ := encrypt(id)

	http.SetCookie(w, &http.Cookie{
		Name:     authCookie,
		Value:    encrypted,
		Path:     "/",
		MaxAge:   authCookieMaxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	return id
}

func encrypt(data string) (string, error) {
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	// Сохраняем: IV + зашифрованные данные
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func decrypt(data string) (string, error) {
	ciphertext, _ := base64.URLEncoding.DecodeString(data)

	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
