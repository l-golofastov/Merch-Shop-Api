package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/l-golofastov/Merch-Shop-Api/internal/config"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/auth"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/buy"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/info"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/sendCoins"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/jwtlib"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/logdiscard"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage/postgres"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type IntegrationSuite struct {
	suite.Suite
	server *httptest.Server
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests")
	}
	suite.Run(t, new(IntegrationSuite))
}

func (s *IntegrationSuite) SetupSuite() {
	log := logdiscard.NewDiscardLogger()

	if err := godotenv.Load("../.env"); err != nil {
		log.Error("Error loading env variables", sl.Err(err))
	}

	dbCfg := config.Postgres{
		Host:     os.Getenv("TEST_POSTGRES_HOST"),
		Port:     os.Getenv("TEST_POSTGRES_PORT"),
		Username: os.Getenv("TEST_POSTGRES_USERNAME"),
		Password: os.Getenv("TEST_POSTGRES_PASSWORD"),
		DBName:   os.Getenv("TEST_POSTGRES_DBNAME"),
	}

	repository, err := postgres.New(dbCfg)
	if err != nil {
		log.Error("failed to init database", sl.Err(err))
	}
	require.NoError(s.T(), err)

	log.Info("starting app tests")

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/api/auth", auth.NewAuthHandler(log, repository))
	router.Get("/api/info", jwtlib.JWTMiddleware(info.NewInfoHandler(log, repository)))
	router.Post("/api/sendCoin", jwtlib.JWTMiddleware(sendCoins.NewCoinSenderHandler(log, repository)))
	router.Get("/api/buy/{item}", jwtlib.JWTMiddleware(buy.NewBuyerHandler(log, repository)))
	router.Get("/api/buy/", jwtlib.JWTMiddleware(buy.NewBuyerHandler(log, repository)))

	// Запускаем тестовый сервер
	s.server = httptest.NewServer(router)
}

func (s *IntegrationSuite) TearDownSuite() {
	s.server.Close()
}

func (s *IntegrationSuite) authUser(username, password string, expectedStatus int, respError error) string {
	t := s.T()

	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, s.server.URL+"/api/auth", bytes.NewReader(jsonBody))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode)

	var retMsg string

	if respError == nil {
		var authResp auth.AuthResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&authResp))
		require.NotEmpty(t, authResp.Token)
		retMsg = authResp.Token
	} else {
		var authResp errresp.ErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&authResp))
		require.NotEmpty(t, authResp.Error)
		retMsg = authResp.Error
	}

	return retMsg
}

func (s *IntegrationSuite) getUserInfo(token string, expectedStatus int, respError error) (info.InfoResponse, errresp.ErrorResponse) {
	t := s.T()

	req, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/info", nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode)

	var infoResp info.InfoResponse
	var errResp errresp.ErrorResponse

	if respError == nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&infoResp))
	} else {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
		require.NotEmpty(t, errResp.Error)
	}

	return infoResp, errResp
}

func (s *IntegrationSuite) buyItem(token, item string, expectedStatus int, respError error) errresp.ErrorResponse {
	t := s.T()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/buy/%s", s.server.URL, item), nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode)

	var errResp errresp.ErrorResponse

	if respError != nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
		require.NotEmpty(t, errResp.Error)
	}

	return errResp
}

func (s *IntegrationSuite) sendCoins(fromToken, toUser string, amount int, expectedStatus int, respError error) errresp.ErrorResponse {
	t := s.T()

	body := map[string]interface{}{
		"toUser": toUser,
		"amount": amount,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, s.server.URL+"/api/sendCoin", bytes.NewReader(jsonBody))
	req.Header.Add("Authorization", "Bearer "+fromToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode)

	var errResp errresp.ErrorResponse

	if respError != nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
		require.NotEmpty(t, errResp.Error)
	}

	return errResp
}

func (s *IntegrationSuite) TestAuthSuccess() {
	s.authUser("user1", "pass1", http.StatusOK, nil)
}

func (s *IntegrationSuite) TestAuthDoubleAuth() {
	token1 := s.authUser("user2", "pass2", http.StatusOK, nil)
	token2 := s.authUser("user2", "pass2", http.StatusOK, nil)
	require.NotEqual(s.T(), token1, token2)
}

func (s *IntegrationSuite) TestAuthIncorrectPassword() {
	s.authUser("user8", "pass8", http.StatusOK, nil)
	resp := s.authUser("user8", "incorrectPassword", http.StatusUnauthorized, errors.New("incorrect password"))
	require.Equal(s.T(), "invalid password", resp)
}

func (s *IntegrationSuite) TestAuthEmptyArgs() {
	s.authUser("", "pass1", http.StatusBadRequest, errors.New("empty username"))
	s.authUser("user1", "", http.StatusBadRequest, errors.New("empty password"))
}

func (s *IntegrationSuite) TestInfoSuccess() {
	token := s.authUser("user3", "pass3", http.StatusOK, nil)
	userInfo, err := s.getUserInfo(token, http.StatusOK, nil)
	require.Equal(s.T(), 1000, userInfo.Coins)
	require.Equal(s.T(), "", err.Error)
	require.Equal(s.T(), 0, len(userInfo.Sent))
	require.Equal(s.T(), 0, len(userInfo.Received))
	require.Equal(s.T(), 0, len(userInfo.Inventory))
}

func (s *IntegrationSuite) TestInfoInvalidToken() {
	_, err := s.getUserInfo("invalid token", http.StatusUnauthorized, errors.New("invalid token"))
	require.Equal(s.T(), err.Error, "invalid token")
}

func (s *IntegrationSuite) TestBuySuccess() {
	token := s.authUser("user4", "pass4", http.StatusOK, nil)
	err := s.buyItem(token, "book", http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)

	userInfo, err := s.getUserInfo(token, http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)
	require.Equal(s.T(), 950, userInfo.Coins)
	require.Equal(s.T(), 1, len(userInfo.Inventory))
	require.Equal(s.T(), 1, userInfo.Inventory[0].Quantity)
	require.Equal(s.T(), "book", userInfo.Inventory[0].Type)
}

func (s *IntegrationSuite) TestBuyNoItemProvided() {
	token := s.authUser("user5", "pass5", http.StatusOK, nil)
	err := s.buyItem(token, "", http.StatusBadRequest, errors.New("no item provided"))
	require.Equal(s.T(), "no item name provided", err.Error)
}

func (s *IntegrationSuite) TestBuyItemNotFound() {
	token := s.authUser("user6", "pass6", http.StatusOK, nil)
	err := s.buyItem(token, "unknownItem", http.StatusBadRequest, storage.ErrItemNotFound)
	require.Equal(s.T(), "item not found", err.Error)
}

func (s *IntegrationSuite) TestBuyNotEnoughCoins() {
	token := s.authUser("user7", "pass7", http.StatusOK, nil)

	err := s.buyItem(token, "pink-hoody", http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)

	err = s.buyItem(token, "pink-hoody", http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)

	err = s.buyItem(token, "pink-hoody", http.StatusBadRequest, storage.ErrNotEnoughCoins)
	require.Equal(s.T(), "not enough coins to buy", err.Error)
}

func (s *IntegrationSuite) TestSendCoinSuccess() {
	token1 := s.authUser("user9", "pass9", http.StatusOK, nil)
	token2 := s.authUser("user10", "pass10", http.StatusOK, nil)

	err := s.sendCoins(token1, "user10", 100, http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)

	infoUser1, err := s.getUserInfo(token1, http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)
	require.Equal(s.T(), 900, infoUser1.Coins)
	require.Equal(s.T(), 1, len(infoUser1.Sent))
	require.Equal(s.T(), "user10", infoUser1.Sent[0].ToUser)
	require.Equal(s.T(), 100, infoUser1.Sent[0].Amount)

	infoUser2, err := s.getUserInfo(token2, http.StatusOK, nil)
	require.Equal(s.T(), "", err.Error)
	require.Equal(s.T(), 1100, infoUser2.Coins)
	require.Equal(s.T(), 1, len(infoUser2.Received))
	require.Equal(s.T(), "user9", infoUser2.Received[0].FromUser)
	require.Equal(s.T(), 100, infoUser2.Received[0].Amount)
}

func (s *IntegrationSuite) TestSendCoinEmptyArgs() {
	token1 := s.authUser("user11", "pass11", http.StatusOK, nil)

	err := s.sendCoins(token1, "", 100, http.StatusBadRequest, errors.New("no username provided"))
	require.Equal(s.T(), "failed to validate request body: not all required fields are provided or amount of coins <= 0", err.Error)
}

func (s *IntegrationSuite) TestSendCoinNegativeAmount() {
	token1 := s.authUser("user12", "pass12", http.StatusOK, nil)

	err := s.sendCoins(token1, "someUser", -10, http.StatusBadRequest, errors.New("no username provided"))
	require.Equal(s.T(), "failed to validate request body: not all required fields are provided or amount of coins <= 0", err.Error)
}

func (s *IntegrationSuite) TestSendCoinUserNotFound() {
	token1 := s.authUser("user13", "pass13", http.StatusOK, nil)

	err := s.sendCoins(token1, "unknownUser", 100, http.StatusBadRequest, storage.ErrUserNotFound)
	require.Equal(s.T(), "user does not exist", err.Error)
}

func (s *IntegrationSuite) TestSendCoinNotEnoughCoins() {
	token1 := s.authUser("user14", "pass14", http.StatusOK, nil)
	s.authUser("user15", "pass15", http.StatusOK, nil)

	err := s.sendCoins(token1, "user15", 1000000, http.StatusBadRequest, storage.ErrNotEnoughCoins)
	require.Equal(s.T(), "not enough coins to send", err.Error)
}
