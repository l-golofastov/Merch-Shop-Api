package jwtlib

import (
	"context"
	"fmt"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"net/http"
	"strings"
)

func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errresp.Error("authorization header required"))

			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errresp.Error("invalid token"))

			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errresp.Error("unable to get token claims"))

			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims["user_id"])
		ctx = context.WithValue(ctx, "username", claims["username"])
		ctx = context.WithValue(ctx, "password_hash", claims["password_hash"])

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
