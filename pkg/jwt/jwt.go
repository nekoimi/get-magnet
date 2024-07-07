package jwt

import (
	jwtlib "github.com/cristalhq/jwt/v5"
	"github.com/google/uuid"
	"github.com/nekoimi/get-magnet/config"
	"time"
)

type Subject interface {
	GetId() string
}

type TokenResult struct {
	Token string `json:"token"`
}

func NewToken(sub Subject) (TokenResult, error) {
	var result TokenResult
	signer, err := jwtlib.NewSignerHS(jwtlib.HS256, []byte(config.Get().JwtSecret))
	if err != nil {
		return result, err
	}

	claims := &jwtlib.RegisteredClaims{
		ID:        uuid.New().String(),
		Subject:   sub.GetId(),
		IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}

	builder := jwtlib.NewBuilder(signer)
	token, err := builder.Build(claims)
	if err != nil {
		return result, err
	}

	result.Token = token.String()
	return result, nil
}
