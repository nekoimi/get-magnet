package util

import "golang.org/x/crypto/bcrypt"

// BcryptEncode Encode 加密
func BcryptEncode(str string) (string, error) {
	bs, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// Check 检测
func Check(encodeStr, rawStr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(encodeStr), []byte(rawStr))
	if err != nil {
		return false
	}
	return true
}
