package cryptoutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

func ID(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func Random() (string, error) {
	bytes := make([]byte, 25)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("error generating random bytes: %w", err)
	}
	token := strings.ToLower(base32.StdEncoding.EncodeToString(bytes))
	return token, nil
}
