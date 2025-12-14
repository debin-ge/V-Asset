package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken 使用 SHA256 哈希 Token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
