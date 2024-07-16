package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// Sha256Encode sha256加密
func Sha256Encode(str string) string {
	s := sha256.New()
	s.Write([]byte(str))
	bs := s.Sum(nil)
	return hex.EncodeToString(bs)
}
