package postgres

import (
	"database/sql"
	"errors"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/datatype/pair"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCheckPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		username      string
		passwordHash  string
		mockBehavior  func()
		expectedID    int
		expectedError error
	}{
		{
			name:         "Valid credentials",
			username:     "user1",
			passwordHash: "correct_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT id FROM users WHERE username = \\$1 AND password_hash = \\$2").
					ExpectQuery().
					WithArgs("user1", "correct_hash").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			},
			expectedID:    1,
			expectedError: nil,
		},
		{
			name:         "Invalid credentials",
			username:     "user1",
			passwordHash: "wrong_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT id FROM users WHERE username = \\$1 AND password_hash = \\$2").
					ExpectQuery().
					WithArgs("user1", "wrong_hash").
					WillReturnError(sql.ErrNoRows)
			},
			expectedID:    -1,
			expectedError: storage.ErrPasswordUnmatched,
		},
		{
			name:         "Database error",
			username:     "user1",
			passwordHash: "correct_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT id FROM users WHERE username = \\$1 AND password_hash = \\$2").
					ExpectQuery().
					WithArgs("user1", "correct_hash").
					WillReturnError(errors.New("database error"))
			},
			expectedID:    -1,
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			id, err := s.CheckPassword(tt.username, tt.passwordHash)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, storage.ErrPasswordUnmatched) {
					assert.True(t, errors.Is(err, storage.ErrPasswordUnmatched))
				} else {
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateUserPasswordHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		userID        int
		newHash       string
		mockBehavior  func()
		expectedError error
	}{
		{
			name:    "Success",
			userID:  1,
			newHash: "new_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("UPDATE users SET password_hash=\\$1 WHERE id=\\$2").
					ExpectExec().
					WithArgs("new_hash", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: nil,
		},
		{
			name:    "No rows affected",
			userID:  999,
			newHash: "new_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("UPDATE users SET password_hash=\\$1 WHERE id=\\$2").
					ExpectExec().
					WithArgs("new_hash", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: nil,
		},
		{
			name:    "Prepare error",
			userID:  1,
			newHash: "new_hash",
			mockBehavior: func() {
				mock.ExpectPrepare("UPDATE users SET password_hash=\\$1 WHERE id=\\$2").
					WillReturnError(errors.New("prepare error"))
			},
			expectedError: errors.New("prepare error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			err := s.UpdateUserPasswordHash(tt.userID, tt.newHash)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetPasswordHashByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		username      string
		mockBehavior  func()
		expectedHash  string
		expectedError error
	}{
		{
			name:     "User exists",
			username: "user1",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT password_hash FROM users WHERE username = \\$1").
					ExpectQuery().
					WithArgs("user1").
					WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow("hash123"))
			},
			expectedHash:  "hash123",
			expectedError: nil,
		},
		{
			name:     "User not found",
			username: "unknown",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT password_hash FROM users WHERE username = \\$1").
					ExpectQuery().
					WithArgs("unknown").
					WillReturnError(sql.ErrNoRows)
			},
			expectedHash:  "",
			expectedError: storage.ErrUserNotFound,
		},
		{
			name:     "Database error",
			username: "user1",
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT password_hash FROM users WHERE username = \\$1").
					ExpectQuery().
					WithArgs("user1").
					WillReturnError(errors.New("connection failed"))
			},
			expectedHash:  "",
			expectedError: errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			hash, err := s.GetPasswordHashByUsername(tt.username)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, storage.ErrUserNotFound) {
					assert.True(t, errors.Is(err, storage.ErrUserNotFound))
				} else {
					assert.Contains(t, err.Error(), tt.expectedError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHash, hash)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserPurchases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		userID        int
		mockBehavior  func()
		expectedItems map[string]int
		expectedError error
	}{
		{
			name:   "Has purchases",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT m.name, p.quantity FROM purchases AS p JOIN merch AS m ON p.merch_id = m.id WHERE p.user_id = \\$1").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"name", "quantity"}).
						AddRow("t-shirt", 2).
						AddRow("cup", 1))
			},
			expectedItems: map[string]int{"t-shirt": 2, "cup": 1},
			expectedError: nil,
		},
		{
			name:   "No purchases",
			userID: 2,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT m.name, p.quantity FROM purchases AS p JOIN merch AS m ON p.merch_id = m.id WHERE p.user_id = \\$1").
					ExpectQuery().
					WithArgs(2).
					WillReturnRows(sqlmock.NewRows([]string{"name", "quantity"}))
			},
			expectedItems: map[string]int{},
			expectedError: nil,
		},
		{
			name:   "Database error",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT m.name, p.quantity FROM purchases AS p JOIN merch AS m ON p.merch_id = m.id WHERE p.user_id = \\$1").
					ExpectQuery().
					WithArgs(1).
					WillReturnError(errors.New("scan error"))
			},
			expectedItems: nil,
			expectedError: errors.New("scan error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			items, err := s.GetUserPurchases(tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedItems, items)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserReceivedCoins(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		userID        int
		mockBehavior  func()
		expectedPairs []pair.Pair[string, int]
		expectedError error
	}{
		{
			name:   "Has received coins",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.from_user_id = u.id WHERE t.to_user_id = \\$1").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"username", "amount"}).
						AddRow("sender1", 100).
						AddRow("sender2", 50))
			},
			expectedPairs: []pair.Pair[string, int]{
				{Fst: "sender1", Snd: 100},
				{Fst: "sender2", Snd: 50},
			},
			expectedError: nil,
		},
		{
			name:   "No received coins",
			userID: 2,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.from_user_id = u.id WHERE t.to_user_id = \\$1").
					ExpectQuery().
					WithArgs(2).
					WillReturnRows(sqlmock.NewRows([]string{"username", "amount"}))
			},
			expectedPairs: []pair.Pair[string, int]{},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			pairs, err := s.GetUserReceivedCoins(tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPairs, pairs)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserSentCoins(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock: %s", err)
	}
	defer db.Close()

	s := &Storage{db: db}

	tests := []struct {
		name          string
		userID        int
		mockBehavior  func()
		expectedPairs []pair.Pair[string, int]
		expectedError error
	}{
		{
			name:   "Has sent coins",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.to_user_id = u.id WHERE t.from_user_id = \\$1").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"username", "amount"}).
						AddRow("receiver1", 200).
						AddRow("receiver2", 150))
			},
			expectedPairs: []pair.Pair[string, int]{
				{Fst: "receiver1", Snd: 200},
				{Fst: "receiver2", Snd: 150},
			},
			expectedError: nil,
		},
		{
			name:   "Error case",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.to_user_id = u.id WHERE t.from_user_id = \\$1").
					ExpectQuery().
					WithArgs(1).
					WillReturnError(errors.New("query error"))
			},
			expectedPairs: nil,
			expectedError: errors.New("query error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			pairs, err := s.GetUserSentCoins(tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPairs, pairs)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storage := &Storage{db: db}

	tests := []struct {
		name         string
		username     string
		passwordHash string
		mockBehavior func()
		expectedID   int
		expectError  bool
	}{
		{
			name:         "Success",
			username:     "testuser",
			passwordHash: "hash",
			mockBehavior: func() {
				mock.ExpectPrepare("INSERT INTO users").
					ExpectQuery().
					WithArgs("testuser", "hash").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			},
			expectedID:  1,
			expectError: false,
		},
		{
			name:         "Prepare Error",
			username:     "testuser",
			passwordHash: "hash",
			mockBehavior: func() {
				mock.ExpectPrepare("INSERT INTO users").
					WillReturnError(errors.New("prepare error"))
			},
			expectedID:  -1,
			expectError: true,
		},
		{
			name:         "Query Error",
			username:     "testuser",
			passwordHash: "hash",
			mockBehavior: func() {
				mock.ExpectPrepare("INSERT INTO users").
					ExpectQuery().
					WithArgs("testuser", "hash").
					WillReturnError(errors.New("query error"))
			},
			expectedID:  -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			id, err := storage.CreateUser(tt.username, tt.passwordHash)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserCoins(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storage := &Storage{db: db}

	tests := []struct {
		name          string
		userID        int
		mockBehavior  func()
		expectedCoins int
		expectError   bool
	}{
		{
			name:   "Success",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(1000))
			},
			expectedCoins: 1000,
			expectError:   false,
		},
		{
			name:   "User Not Found",
			userID: 1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(1).
					WillReturnError(sql.ErrNoRows)
			},
			expectedCoins: -1,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			coins, err := storage.GetUserCoins(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCoins, coins)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSendCoins(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &Storage{db: db}

	tests := []struct {
		name          string
		username      string
		userID        int
		amount        int
		mockBehavior  func()
		expectError   bool
		expectedError error
	}{
		{
			name:     "Success",
			username: "receiver",
			userID:   1,
			amount:   100,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(1000))

				mock.ExpectPrepare("SELECT id FROM users").
					ExpectQuery().
					WithArgs("receiver").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(2).
					WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(500))

				mock.ExpectPrepare("INSERT INTO transfers").
					ExpectExec().
					WithArgs(1, 2, 100).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectPrepare("UPDATE users SET coins").
					ExpectExec().
					WithArgs(900, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectPrepare("UPDATE users SET coins").
					ExpectExec().
					WithArgs(600, 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
		{
			name:     "Not Enough Coins",
			username: "receiver",
			userID:   1,
			amount:   100,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(50))
			},
			expectError:   true,
			expectedError: storage.ErrNotEnoughCoins,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			err := repo.SendCoins(tt.username, tt.userID, tt.amount)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.True(t, errors.Is(err, tt.expectedError))
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBuyItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	storage := &Storage{db: db}

	tests := []struct {
		name          string
		itemName      string
		userID        int
		mockBehavior  func()
		expectError   bool
		expectedError error
	}{
		{
			name:     "Success - New Item",
			itemName: "t-shirt",
			userID:   1,
			mockBehavior: func() {
				mock.ExpectPrepare("SELECT price FROM merch").
					ExpectQuery().
					WithArgs("t-shirt").
					WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(80))

				mock.ExpectPrepare("SELECT coins FROM users").
					ExpectQuery().
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"coins"}).AddRow(1000))

				mock.ExpectPrepare("SELECT id FROM purchases").
					ExpectQuery().
					WithArgs(1, "t-shirt").
					WillReturnError(sql.ErrNoRows)

				mock.ExpectPrepare("INSERT INTO purchases").
					ExpectExec().
					WithArgs(1, "t-shirt").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectPrepare("UPDATE users SET coins").
					ExpectExec().
					WithArgs(920, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			err := storage.BuyItem(tt.itemName, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.True(t, errors.Is(err, tt.expectedError))
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
