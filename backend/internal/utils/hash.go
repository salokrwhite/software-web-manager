package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/fnv"
)

func HashPercent(input string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(input))
	return int(h.Sum32() % 100)
}

func SHA256Hex(input []byte) string {
	h := sha256.New()
	_, _ = h.Write(input)
	return hex.EncodeToString(h.Sum(nil))
}

