package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/l-golofastov/Merch-Shop-Api/internal/config"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/datatype/pair"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(cfg config.Postgres) (*Storage, error) {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	usersStmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			coins INTEGER NOT NULL DEFAULT 1000 CHECK (coins >= 0),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = usersStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	merchStmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS merch (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		price INTEGER NOT NULL CHECK (price > 0)
		);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = merchStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	checkMerchStmt, err := db.Prepare(`
		SELECT * FROM merch;
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = checkMerchStmt.Exec()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			merchInsertStmt, err := db.Prepare(`
				INSERT INTO merch (name, price) VALUES
				('t-shirt', 80), ('cup', 20), ('book', 50),
				('pen', 10), ('powerbank', 200), ('hoody', 300),
				('umbrella', 200), ('socks', 10), ('wallet', 50),
				('pink-hoody', 500);
			`)

			if err != nil {
				return nil, fmt.Errorf("%s %w", op, err)
			}

			_, err = merchInsertStmt.Exec()
			if err != nil {
				return nil, fmt.Errorf("%s %w", op, err)
			}
		} else {
			return nil, fmt.Errorf("%s %w", op, err)
		}
	}

	purchasesStmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS purchases (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			merch_id INTEGER REFERENCES merch(id) ON DELETE CASCADE,
			quantity INTEGER NOT NULL CHECK (quantity > 0),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = purchasesStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	purchasesIndexStmt, err := db.Prepare(`
		CREATE INDEX IF NOT EXISTS idx_purchases_user ON purchases(user_id);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = purchasesIndexStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	transfersStmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS transfers (
			id SERIAL PRIMARY KEY,
			from_user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			to_user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			amount INTEGER NOT NULL CHECK (amount > 0),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = transfersStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	indexFromStmt, err := db.Prepare(`
		CREATE INDEX IF NOT EXISTS idx_transfers_from ON transfers(from_user_id);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = indexFromStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	indexToStmt, err := db.Prepare(`
		CREATE INDEX IF NOT EXISTS idx_transfers_to ON transfers(to_user_id);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = indexToStmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) CreateUser(username, passwordHash string) (int, error) {
	const op = "storage.postgres.CreateUser"

	var userId int

	stmt, err := s.db.Prepare(`INSERT INTO users (username, password_hash, coins) VALUES ($1, $2, 1000) RETURNING id;`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	err = stmt.QueryRow(username, passwordHash).Scan(&userId)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return userId, nil
}

func (s *Storage) CheckPassword(username, passwordHash string) (int, error) {
	const op = "storage.postgres.CheckPassword"

	var userId int

	stmt, err := s.db.Prepare(`SELECT id FROM users WHERE username = $1 AND password_hash = $2;`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	err = stmt.QueryRow(username, passwordHash).Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, storage.ErrPasswordUnmatched
		}
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return userId, nil
}

func (s *Storage) UpdateUserPasswordHash(id int, hash string) error {
	const op = "storage.postgres.UpdateUserPasswordHash"

	stmt, err := s.db.Prepare(`UPDATE users SET password_hash=$1 WHERE id=$2;`)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	_, err = stmt.Exec(hash, id)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	return nil
}

func (s *Storage) FindUserByUsername(username string) (int, error) {
	const op = "storage.postgres.FindUserByUsername"

	var userId int

	stmt, err := s.db.Prepare(`SELECT id FROM users WHERE username = $1;`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, storage.ErrUserNotFound
		}
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return userId, nil
}

func (s *Storage) GetPasswordHashByUsername(username string) (string, error) {
	const op = "storage.postgres.GetPasswordHashByUsername"

	var hash string

	stmt, err := s.db.Prepare(`SELECT password_hash FROM users WHERE username = $1;`)
	if err != nil {
		return "", fmt.Errorf("%s %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrUserNotFound
		}
		return "", fmt.Errorf("%s %w", op, err)
	}

	return hash, nil
}

func (s *Storage) GetUserCoins(id int) (int, error) {
	const op = "storage.postgres.GetUserCoins"

	var coins int

	stmt, err := s.db.Prepare(`SELECT coins FROM users WHERE id = $1;`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	err = stmt.QueryRow(id).Scan(&coins)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return coins, nil
}

func (s *Storage) GetUserPurchases(id int) (map[string]int, error) {
	const op = "storage.postgres.GetUserPurchases"

	inventory := make(map[string]int)

	stmt, err := s.db.Prepare(`
		SELECT m.name, p.quantity FROM purchases AS p JOIN merch AS m ON p.merch_id = m.id
		WHERE p.user_id = $1;
	`)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	rows, err := stmt.Query(id)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	for rows.Next() {
		var name string
		var quantity int

		err = rows.Scan(&name, &quantity)
		if err != nil {
			return nil, fmt.Errorf("%s %w", op, err)
		}
		inventory[name] = quantity
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return inventory, nil
}

func (s *Storage) GetUserReceivedCoins(id int) ([]pair.Pair[string, int], error) {
	const op = "storage.postgres.GetUserReceivedCoins"

	getFromSlice := make([]pair.Pair[string, int], 0)

	stmt, err := s.db.Prepare(`
		SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.from_user_id = u.id
		WHERE t.to_user_id = $1;
	`)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	rows, err := stmt.Query(id)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	for rows.Next() {
		var usernameFrom string
		var quantity int

		err = rows.Scan(&usernameFrom, &quantity)
		if err != nil {
			return nil, fmt.Errorf("%s %w", op, err)
		}
		getFromSlice = append(getFromSlice, pair.Pair[string, int]{Fst: usernameFrom, Snd: quantity})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return getFromSlice, nil
}

func (s *Storage) GetUserSentCoins(id int) ([]pair.Pair[string, int], error) {
	const op = "storage.postgres.GetUserSentCoins"

	sentToSlice := make([]pair.Pair[string, int], 0)

	stmt, err := s.db.Prepare(`
		SELECT u.username, t.amount FROM transfers AS t JOIN users AS u ON t.to_user_id = u.id
		WHERE t.from_user_id = $1;
	`)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	rows, err := stmt.Query(id)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	for rows.Next() {
		var usernameTo string
		var quantity int

		err = rows.Scan(&usernameTo, &quantity)
		if err != nil {
			return nil, fmt.Errorf("%s %w", op, err)
		}
		sentToSlice = append(sentToSlice, pair.Pair[string, int]{Fst: usernameTo, Snd: quantity})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return sentToSlice, nil
}

func (s *Storage) SendCoins(username string, id, amount int) error {
	const op = "storage.postgres.SendCoins"

	coinsFrom, err := s.GetUserCoins(id)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	if coinsFrom < amount {
		return storage.ErrNotEnoughCoins
	}

	toId, err := s.FindUserByUsername(username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return storage.ErrUserNotFound
		}
		return fmt.Errorf("%s %w", op, err)
	}

	coinsTo, err := s.GetUserCoins(toId)

	coinsFromNew := coinsFrom - amount
	coinsToNew := coinsTo + amount

	insertTransfersStmt, err := s.db.Prepare(`
		INSERT INTO transfers (from_user_id, to_user_id, amount) VALUES ($1, $2, $3);
	`)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	_, err = insertTransfersStmt.Exec(id, toId, amount)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	updateSenderStmt, err := s.db.Prepare(`
		UPDATE users SET coins = $1 WHERE id = $2;
	`)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	_, err = updateSenderStmt.Exec(coinsFromNew, id)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	updateReceiverStmt, err := s.db.Prepare(`
		UPDATE users SET coins = $1 WHERE id = $2;
	`)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	_, err = updateReceiverStmt.Exec(coinsToNew, toId)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	return nil
}

func (s *Storage) GetItemPrice(itemName string) (int, error) {
	const op = "storage.postgres.GetItemPrice"

	stmt, err := s.db.Prepare(`SELECT price FROM merch WHERE name = $1;`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	var price int

	err = stmt.QueryRow(itemName).Scan(&price)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, storage.ErrItemNotFound
		}
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return price, nil
}

func (s *Storage) CheckIfUserAlreadyHasItem(itemName string, userId int) (int, error) {
	const op = "storage.postgres.CheckIfUserAlreadyHasItem"

	stmt, err := s.db.Prepare(`
		SELECT id FROM purchases WHERE user_id = $1
		AND merch_id = (SELECT id FROM merch WHERE name = $2);
	`)
	if err != nil {
		return -1, fmt.Errorf("%s %w", op, err)
	}

	var id int

	err = stmt.QueryRow(userId, itemName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, nil
		}
		return -1, fmt.Errorf("%s %w", op, err)
	}

	return id, nil
}

func (s *Storage) BuyItem(itemName string, userId int) error {
	const op = "storage.postgres.BuyItem"

	price, err := s.GetItemPrice(itemName)
	if err != nil {
		if errors.Is(err, storage.ErrItemNotFound) {
			return storage.ErrItemNotFound
		}
		return fmt.Errorf("%s %w", op, err)
	}

	userCoins, err := s.GetUserCoins(userId)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	if userCoins < price {
		return storage.ErrNotEnoughCoins
	}

	userCoinsNew := userCoins - price

	lineId, err := s.CheckIfUserAlreadyHasItem(itemName, userId)
	if err != nil {
		return fmt.Errorf("%s %w", op, err)
	}

	if lineId != -1 {
		updatePurchasesStmt, err := s.db.Prepare(`
			UPDATE purchases SET quantity = quantity + 1 WHERE id = $1;
		`)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		_, err = updatePurchasesStmt.Exec(lineId)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		updateUsersStmt, err := s.db.Prepare(`
			UPDATE users SET coins = $1 WHERE id = $2;
		`)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		_, err = updateUsersStmt.Exec(userCoinsNew, userId)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}
	} else {
		insertPurchasesStmt, err := s.db.Prepare(`
			INSERT INTO purchases (user_id, merch_id, quantity) VALUES ($1, (SELECT id FROM merch WHERE name = $2), 1);
		`)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		_, err = insertPurchasesStmt.Exec(userId, itemName)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		updateUsersStmt, err := s.db.Prepare(`
			UPDATE users SET coins = $1 WHERE id = $2;
		`)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}

		_, err = updateUsersStmt.Exec(userCoinsNew, userId)
		if err != nil {
			return fmt.Errorf("%s %w", op, err)
		}
	}

	return nil
}
