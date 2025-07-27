package jwt

import (
	"encoding/json"
	"errors"
	jwtlib "github.com/cristalhq/jwt/v5"
	"github.com/google/uuid"
	"time"
)

var (
	jwtSecret        string
	TokenExpireError = errors.New("认证信息已过期")
)

func SetSecret(secret string) {
	jwtSecret = secret
}

type Subject interface {
	GetId() string
}

type TokenResult struct {
	Token string `json:"token"`
}

func NewToken(sub Subject) (TokenResult, error) {
	var result TokenResult
	signer, err := jwtlib.NewSignerHS(jwtlib.HS256, []byte(jwtSecret))
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

func ParseToken(token string) (string, error) {
	verifier, err := jwtlib.NewVerifierHS(jwtlib.HS256, []byte(jwtSecret))
	if err != nil {
		return "", err
	}

	t, err := jwtlib.Parse([]byte(token), verifier)
	if err != nil {
		return "", err
	}

	err = verifier.Verify(t)
	if err != nil {
		return "", err
	}

	var claims jwtlib.RegisteredClaims
	err = json.Unmarshal(t.Claims(), &claims)
	if err != nil {
		return "", err
	}

	if !claims.IsValidExpiresAt(time.Now()) {
		return "", TokenExpireError
	}

	return claims.Subject, nil
}
