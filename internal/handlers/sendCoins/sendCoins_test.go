package sendCoins

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/sendCoins/mocks"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/jwtlib"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/logdiscard"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCoinSenderHandler(t *testing.T) {
	userId := 4
	username := "testuser"
	password := "testpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passwordHash := string(hashedPassword)
	testJwtToken, _ := jwtlib.GenerateJWT(userId, username, passwordHash)
	bearer := "Bearer " + testJwtToken

	cases := []struct {
		name             string
		userId           int
		checkPasswordErr error
		userIdUnmatched  bool
		toUser           string
		amount           int
		sendCoinsErr     error
		respError        string
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
			name:         "Not enough coins",
			userId:       userId,
			toUser:       "someone",
			amount:       10,
			sendCoinsErr: storage.ErrNotEnoughCoins,
			respError:    "not enough coins to send",
		},
		{
			name:         "User not found",
			userId:       userId,
			toUser:       "someone",
			amount:       10,
			sendCoinsErr: storage.ErrUserNotFound,
			respError:    "user does not exist",
		},
		{
			name:         "User not found",
			userId:       userId,
			toUser:       "someone",
			amount:       10,
			sendCoinsErr: errors.New("error sending coins"),
			respError:    "internal server error",
		},
		{
			name:   "Successful request",
			userId: userId,
			toUser: "someone",
			amount: 10,
		},
		{
			name:      "To user empty",
			userId:    userId,
			amount:    10,
			respError: "failed to validate request body: not all required fields are provided or amount of coins <= 0",
		},
		{
			name:      "Negative amount of coins",
			userId:    userId,
			toUser:    "someone",
			amount:    -10,
			respError: "failed to validate request body: not all required fields are provided or amount of coins <= 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			coinSenderHandlerMock := mocks.NewCoinSender(t)

			coinSenderHandlerMock.On("CheckPassword", username, passwordHash).
				Return(tc.userId, tc.checkPasswordErr).
				Once()

			if tc.checkPasswordErr == nil && !tc.userIdUnmatched && tc.toUser != "" && tc.amount > 0 {
				coinSenderHandlerMock.On("SendCoins", tc.toUser, userId, tc.amount).
					Return(tc.sendCoinsErr).
					Once()
			}

			handler := jwtlib.JWTMiddleware(NewCoinSenderHandler(logdiscard.NewDiscardLogger(), coinSenderHandlerMock))

			input := fmt.Sprintf(`{"toUser":"%s","amount":%d}`, tc.toUser, tc.amount)

			req, err := http.NewRequest(http.MethodPost, "/api/sendCoin", bytes.NewReader([]byte(input)))
			req.Header.Add("Authorization", bearer)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			body := rr.Body.String()

			if tc.checkPasswordErr != nil || tc.userIdUnmatched {
				if errors.Is(tc.checkPasswordErr, storage.ErrPasswordUnmatched) || tc.userIdUnmatched {
					require.Equal(t, http.StatusUnauthorized, rr.Code)
				} else {
					require.Equal(t, http.StatusInternalServerError, rr.Code)
				}
			} else if tc.toUser == "" || tc.amount <= 0 || tc.sendCoinsErr != nil {
				if tc.sendCoinsErr != nil &&
					!errors.Is(tc.sendCoinsErr, storage.ErrNotEnoughCoins) &&
					!errors.Is(tc.sendCoinsErr, storage.ErrUserNotFound) {
					require.Equal(t, http.StatusInternalServerError, rr.Code)
				} else {
					require.Equal(t, http.StatusBadRequest, rr.Code)
				}

				var resp errresp.ErrorResponse
				require.NoError(t, json.Unmarshal([]byte(body), &resp))
				require.Equal(t, resp.Error, tc.respError)

			} else {
				require.Equal(t, http.StatusOK, rr.Code)
				require.Empty(t, body)
			}
		})
	}
}
