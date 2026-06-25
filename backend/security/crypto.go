package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
)

// devFallbackKey is used only when LDAP_ENCRYPTION_KEY is not set (dev/test).
// Must never be used in production.
const devFallbackKey = "dev-fallback-32bytekey!NOTP ROD"

// ldapKey returns the 32-byte AES-256 key from env LDAP_ENCRYPTION_KEY.
// Accepts either a plain 32-byte string or a base64-encoded 32-byte value.
func ldapKey() ([]byte, error) {
	raw := os.Getenv("LDAP_ENCRYPTION_KEY")
	if raw == "" {
		log.Println("[WARN] LDAP_ENCRYPTION_KEY tidak diset — menggunakan dev key. Jangan gunakan di production!")
		return []byte(devFallbackKey), nil
	}

	// Try base64 first (preferred for env vars with binary-safe transport)
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	// Fallback: treat as raw string
	b := []byte(raw)
	if len(b) != 32 {
		return nil, errors.New("LDAP_ENCRYPTION_KEY harus 32 byte (256-bit); generate dengan: openssl rand -base64 32 | head -c 44")
	}
	return b, nil
}

// EncryptLDAP encrypts plaintext using AES-256-GCM.
// Returns a base64-encoded string: nonce (12 bytes) + ciphertext + GCM tag.
func EncryptLDAP(plaintext string) (string, error) {
	key, err := ldapKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptLDAP decrypts a base64-encoded AES-256-GCM ciphertext produced by EncryptLDAP.
func DecryptLDAP(encoded string) (string, error) {
	key, err := ldapKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("format ciphertext tidak valid")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext terlalu pendek")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("dekripsi gagal: password tidak valid atau key salah")
	}
	return string(plaintext), nil
}
