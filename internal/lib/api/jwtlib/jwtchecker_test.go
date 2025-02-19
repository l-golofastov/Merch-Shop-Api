package jwtlib

import (
	"github.com/golang-jwt/jwt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTMiddleware(t *testing.T) {

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(string)
		username := r.Context().Value("username").(string)
		passwordHash := r.Context().Value("password_hash").(string)

		w.Header().Set("X-User-ID", userID)
		w.Header().Set("X-Username", username)
		w.Header().Set("X-PasswordHash", passwordHash)
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		expectHeaders  map[string]string
	}{
		{
			name: "No Authorization header",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api", nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token format",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api", nil)
				req.Header.Set("Authorization", "InvalidToken")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Expired token",
			setupRequest: func() *http.Request {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":       "123",
					"username":      "testuser",
					"password_hash": "hash",
					"exp":           time.Now().Add(-1 * time.Hour).Unix(),
				})
				signedToken, _ := token.SignedString(jwtSecret)

				req := httptest.NewRequest(http.MethodGet, "/api", nil)
				req.Header.Set("Authorization", "Bearer "+signedToken)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Valid token",
			setupRequest: func() *http.Request {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id":       "123",
					"username":      "testuser",
					"password_hash": "testhash",
				})
				signedToken, _ := token.SignedString(jwtSecret)

				req := httptest.NewRequest(http.MethodGet, "/api", nil)
				req.Header.Set("Authorization", "Bearer "+signedToken)
				return req
			},
			expectedStatus: http.StatusOK,
			expectHeaders: map[string]string{
				"X-User-ID":      "123",
				"X-Username":     "testuser",
				"X-PasswordHash": "testhash",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := JWTMiddleware(mockHandler)

			req := tc.setupRequest()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			for hdr, expectedValue := range tc.expectHeaders {
				if actual := rr.Header().Get(hdr); actual != expectedValue {
					t.Errorf("header %s mismatch: got %v want %v",
						hdr, actual, expectedValue)
				}
			}
		})
	}
}
