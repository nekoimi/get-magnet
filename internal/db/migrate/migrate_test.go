package migrate

import (
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"testing"
)

func TestEncodePassword(t *testing.T) {
	sha256Str := util.Sha256Encode(defaultAdminPassword)
	encode, err := util.BcryptEncode(sha256Str)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("Default Password: ", encode)
}
