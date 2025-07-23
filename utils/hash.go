package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(bs []byte) string {
	bytes := sha256.Sum256(bs)
	return hex.EncodeToString(bytes[:])
}
