package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/auth/mocks"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/logdiscard"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthHandler(t *testing.T) {
	cases := []struct {
		name          string
		username      string
		password      string
		findUserId    int
		findUserErr   error
		createUserErr error
		getHashErr    error
		updateHashErr error
		passwordValid bool
		token         string
		respError     string
	}{
		{
			name:        "SuccessCreatingNewUser",
			username:    "testuser",
			password:    "testpassword",
			findUserId:  -1,
			findUserErr: storage.ErrUserNotFound,
		},
		{
			name:        "ErrorFindingUser",
			username:    "testuser",
			password:    "testpassword",
			findUserId:  -1,
			findUserErr: errors.New("finding user storage error"),
			respError:   "internal server error",
		},
		{
			name:          "ErrorCreatingUser",
			username:      "testuser",
			password:      "testpassword",
			findUserId:    -1,
			findUserErr:   storage.ErrUserNotFound,
			createUserErr: errors.New("creating user storage error"),
			respError:     "internal server error",
		},
		{
			name:      "EmptyUsername",
			password:  "testpassword",
			respError: "not all required fields are provided",
		},
		{
			name:      "EmptyPassword",
			username:  "testuser",
			respError: "not all required fields are provided",
		},
		{
			name:      "EmptyUsernameAndPassword",
			respError: "not all required fields are provided",
		},
		{
			name:          "SuccessUpdatingUserPasswordHash",
			username:      "testuser",
			password:      "testpassword",
			findUserId:    1,
			passwordValid: true,
		},
		{
			name:          "ErrorComparingHashAndPassword",
			username:      "testuser",
			password:      "testpassword",
			findUserId:    1,
			passwordValid: false,
			respError:     "invalid password",
		},
		{
			name:          "ErrorGettingPasswordHash",
			username:      "testuser",
			password:      "testpassword",
			findUserId:    1,
			passwordValid: false,
			getHashErr:    errors.New("getting hash storage error"),
			respError:     "internal server error",
		},
		{
			name:          "ErrorUpdatingPasswordHash",
			username:      "testuser",
			password:      "testpassword",
			findUserId:    1,
			passwordValid: true,
			updateHashErr: errors.New("updating password hash storage error"),
			respError:     "internal server error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authHandlerMock := mocks.NewAuthHandler(t)

			if tc.username != "" && tc.password != "" {
				var passwordHash string
				if tc.passwordValid {
					hash, _ := bcrypt.GenerateFromPassword([]byte(tc.password), bcrypt.DefaultCost)
					passwordHash = string(hash)
				} else {
					passwordHash = string(mock.AnythingOfType("string"))
				}

				authHandlerMock.On("FindUserByUsername", tc.username).
					Return(tc.findUserId, tc.findUserErr).
					Once()

				if tc.findUserId != -1 {
					authHandlerMock.On("GetPasswordHashByUsername", tc.username).
						Return(passwordHash, tc.getHashErr).
						Once()

					if tc.getHashErr == nil && tc.passwordValid {
						authHandlerMock.On("UpdateUserPasswordHash", tc.findUserId, mock.AnythingOfType("string")).
							Return(tc.updateHashErr).
							Once()
					}
				} else if errors.Is(tc.findUserErr, storage.ErrUserNotFound) {
					if tc.createUserErr != nil {
						authHandlerMock.On("CreateUser", tc.username, mock.AnythingOfType("string")).
							Return(-1, tc.createUserErr).
							Once()
					} else {
						authHandlerMock.On("CreateUser", tc.username, mock.AnythingOfType("string")).
							Return(1, tc.createUserErr).
							Once()
					}
				}
			}

			handler := NewAuthHandler(logdiscard.NewDiscardLogger(), authHandlerMock)

			input := fmt.Sprintf(`{"username":"%s","password":"%s"}`, tc.username, tc.password)

			req, err := http.NewRequest(http.MethodPost, "/api/auth", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			body := rr.Body.String()

			if tc.username == "" || tc.password == "" {
				require.Equal(t, http.StatusBadRequest, rr.Code)

				var resp errresp.ErrorResponse

				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.Equal(t, resp.Error, tc.respError)
			} else if tc.findUserErr != nil && !errors.Is(tc.findUserErr, storage.ErrUserNotFound) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)

				var resp errresp.ErrorResponse

				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.Equal(t, resp.Error, tc.respError)
			} else if tc.findUserId != -1 {
				if tc.getHashErr != nil {
					require.Equal(t, http.StatusInternalServerError, rr.Code)

					var resp errresp.ErrorResponse

					require.NoError(t, json.Unmarshal([]byte(body), &resp))
					require.Equal(t, resp.Error, tc.respError)
				} else if !tc.passwordValid {
					require.Equal(t, http.StatusUnauthorized, rr.Code)

					var resp errresp.ErrorResponse

					require.NoError(t, json.Unmarshal([]byte(body), &resp))
					require.Equal(t, resp.Error, tc.respError)
				} else if tc.updateHashErr != nil {
					require.Equal(t, http.StatusInternalServerError, rr.Code)

					var resp errresp.ErrorResponse

					require.NoError(t, json.Unmarshal([]byte(body), &resp))
					require.Equal(t, resp.Error, tc.respError)
				} else {
					require.Equal(t, http.StatusOK, rr.Code)

					var resp AuthResponse

					require.NoError(t, json.Unmarshal([]byte(body), &resp))
					require.NotEmpty(t, resp.Token)
				}
			} else if tc.createUserErr != nil {
				require.Equal(t, http.StatusInternalServerError, rr.Code)

				var resp errresp.ErrorResponse

				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.Equal(t, resp.Error, tc.respError)
			} else {
				require.Equal(t, http.StatusOK, rr.Code)

				var resp AuthResponse

				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.NotEmpty(t, resp.Token)
			}
		})
	}
}
