package cryptoutils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

const (
	hashCost   = bcrypt.DefaultCost
	apiKeySize = 32
)

func HashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	return hash, err
}

func CompareHashAndPassword(hashedPassword []byte, password string) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
}

func Sha256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func XApiKey() string {
	b, err := RandomBytes(apiKeySize)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
