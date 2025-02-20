package buy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/buy/mocks"
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

func TestBuyerHandler(t *testing.T) {
	userId := 4
	username := "testuser"
	password := "testpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passwordHash := string(hashedPassword)
	testJwtToken, _ := jwtlib.GenerateJWT(userId, username, passwordHash)
	bearer := "Bearer " + testJwtToken
	itemName := "book"

	cases := []struct {
		name             string
		userId           int
		itemName         string
		checkPasswordErr error
		userIdUnmatched  bool
		buyErr           error
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
			name:      "No item provided",
			userId:    userId,
			respError: "no item name provided",
		},
		{
			name:      "Item not found",
			userId:    userId,
			itemName:  "strangeItem",
			buyErr:    storage.ErrItemNotFound,
			respError: "item not found",
		},
		{
			name:      "Not enough coins",
			userId:    userId,
			itemName:  itemName,
			buyErr:    storage.ErrNotEnoughCoins,
			respError: "not enough coins to buy",
		},
		{
			name:     "Successful request",
			userId:   userId,
			itemName: itemName,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buyerHandlerMock := mocks.NewBuyer(t)

			buyerHandlerMock.On("CheckPassword", username, passwordHash).
				Return(tc.userId, tc.checkPasswordErr).
				Once()

			if tc.checkPasswordErr == nil && !tc.userIdUnmatched && tc.itemName != "" {
				buyerHandlerMock.On("BuyItem", tc.itemName, userId).
					Return(tc.buyErr).
					Once()
			}

			r := chi.NewRouter()
			r.Get("/api/buy/", jwtlib.JWTMiddleware(NewBuyerHandler(logdiscard.NewDiscardLogger(), buyerHandlerMock)))
			r.Get("/api/buy/{item}", jwtlib.JWTMiddleware(NewBuyerHandler(logdiscard.NewDiscardLogger(), buyerHandlerMock)))

			url := fmt.Sprintf("/api/buy/%s", tc.itemName)

			req, err := http.NewRequest(http.MethodGet, url, nil)
			req.Header.Add("Authorization", bearer)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			body := rr.Body.String()

			if tc.checkPasswordErr != nil || tc.userIdUnmatched {
				if errors.Is(tc.checkPasswordErr, storage.ErrPasswordUnmatched) || tc.userIdUnmatched {
					require.Equal(t, http.StatusUnauthorized, rr.Code)
				} else {
					require.Equal(t, http.StatusInternalServerError, rr.Code)
				}
			} else if tc.itemName == "" || tc.buyErr != nil {
				if tc.itemName == "" ||
					errors.Is(tc.buyErr, storage.ErrItemNotFound) ||
					errors.Is(tc.buyErr, storage.ErrNotEnoughCoins) {
					require.Equal(t, http.StatusBadRequest, rr.Code)
				} else {
					require.Equal(t, http.StatusInternalServerError, rr.Code)
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
