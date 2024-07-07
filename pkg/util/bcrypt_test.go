package util

import "testing"

func TestBcryptEncode(t *testing.T) {
	encode, err := BcryptEncode("123456")
	if err != nil {
		return
	}
	t.Log(encode)
}
