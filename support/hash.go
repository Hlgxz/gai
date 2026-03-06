package support

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hash returns a bcrypt hash of the password.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckHash compares a password with a bcrypt hash.
func CheckHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Sha256 returns the SHA-256 hex digest of the input.
func Sha256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// RandomString generates a cryptographically secure random hex string of n bytes.
func RandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("gai: failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}
