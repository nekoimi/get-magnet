package util

import "testing"

func TestSha256Encode(t *testing.T) {
	t.Log(Sha256Encode("123456"))
}
