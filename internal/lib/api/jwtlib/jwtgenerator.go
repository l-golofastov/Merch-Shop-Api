package jwtlib

import (
	"github.com/golang-jwt/jwt"
	"time"
)

var jwtSecret = []byte("81b3b7186fb2bf989eba8f76f3c98040e19f2b8b4b3730d5593b2a88e2f9ec11")

func GenerateJWT(userId int, username, passwordHash string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":       userId,
		"username":      username,
		"password_hash": passwordHash,
		"expire":        time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtSecret)
}
