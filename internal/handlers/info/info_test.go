package info

import (
	"encoding/json"
	"errors"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/info/mocks"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/jwtlib"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/datatype/pair"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/logdiscard"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfoHandler(t *testing.T) {
	userId := 4
	username := "testuser"
	password := "testpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passwordHash := string(hashedPassword)
	testJwtToken, _ := jwtlib.GenerateJWT(userId, username, passwordHash)
	bearer := "Bearer " + testJwtToken

	cases := []struct {
		name                 string
		userId               int
		checkPasswordErr     error
		userIdUnmatched      bool
		getUserCoinsErr      error
		getUserPurchasesErr  error
		getUserReceivedCoins error
		getUserSentCoinsErr  error
		respError            string
	}{
		{
			name:             "Password unmatched",
			userId:           userId,
			checkPasswordErr: storage.ErrPasswordUnmatched,
			respError:        "invalid token claims: password",
		},
		{
			name:             "Error getting password from storage",
			userId:           userId,
			checkPasswordErr: errors.New("error getting password from storage"),
			respError:        "internal server error",
		},
		{
			name:            "User id unmatched",
			userId:          0,
			userIdUnmatched: true,
			respError:       "invalid token claims: id",
		},
		{
			name:            "Error getting user coins from storage",
			userId:          userId,
			getUserCoinsErr: errors.New("error getting user coins from storage"),
			respError:       "internal server error",
		},
		{
			name:                "Error getting user purchases from storage",
			userId:              userId,
			getUserPurchasesErr: errors.New("error getting user purchases from storage"),
			respError:           "internal server error",
		},
		{
			name:                 "Error getting user received coins from storage",
			userId:               userId,
			getUserReceivedCoins: errors.New("error getting user received coins from storage"),
			respError:            "internal server error",
		},
		{
			name:                "Error getting user sent coins from storage",
			userId:              userId,
			getUserSentCoinsErr: errors.New("error getting user sent coins from storage"),
			respError:           "internal server error",
		},
		{
			name:   "Successful request",
			userId: userId,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			infoHandlerMock := mocks.NewInfoHandler(t)

			infoHandlerMock.On("CheckPassword", username, passwordHash).
				Return(tc.userId, tc.checkPasswordErr).
				Once()

			if tc.checkPasswordErr == nil && !tc.userIdUnmatched {
				infoHandlerMock.On("GetUserCoins", tc.userId).
					Return(0, tc.getUserCoinsErr).
					Once()

				if tc.getUserCoinsErr == nil {
					infoHandlerMock.On("GetUserPurchases", tc.userId).
						Return(map[string]int{}, tc.getUserPurchasesErr).
						Once()

					if tc.getUserPurchasesErr == nil {
						infoHandlerMock.On("GetUserReceivedCoins", tc.userId).
							Return([]pair.Pair[string, int]{}, tc.getUserReceivedCoins).
							Once()

						if tc.getUserReceivedCoins == nil {
							infoHandlerMock.On("GetUserSentCoins", tc.userId).
								Return([]pair.Pair[string, int]{}, tc.getUserSentCoinsErr).
								Once()
						}
					}
				}
			}

			handler := jwtlib.JWTMiddleware(NewInfoHandler(logdiscard.NewDiscardLogger(), infoHandlerMock))

			req, err := http.NewRequest(http.MethodGet, "/api/info", nil)
			req.Header.Add("Authorization", bearer)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			body := rr.Body.String()

			if tc.checkPasswordErr != nil ||
				tc.userIdUnmatched ||
				tc.getUserCoinsErr != nil ||
				tc.getUserPurchasesErr != nil ||
				tc.getUserReceivedCoins != nil ||
				tc.getUserSentCoinsErr != nil {
				if errors.Is(tc.checkPasswordErr, storage.ErrPasswordUnmatched) || tc.userIdUnmatched {
					require.Equal(t, http.StatusUnauthorized, rr.Code)
				} else {
					require.Equal(t, http.StatusInternalServerError, rr.Code)
				}
				var resp errresp.ErrorResponse
				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.Equal(t, resp.Error, tc.respError)
			} else {
				require.Equal(t, http.StatusOK, rr.Code)

				var resp InfoResponse
				require.NoError(t, json.Unmarshal([]byte(body), &resp))
			}
		})
	}
}
