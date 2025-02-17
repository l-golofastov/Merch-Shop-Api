package jwtlib

import (
	"github.com/golang-jwt/jwt"
	"os"
	"time"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

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
