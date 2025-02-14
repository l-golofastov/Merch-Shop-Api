package postgres

import (
	"database/sql"
	"fmt"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	prepare, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		coins INTEGER NOT NULL DEFAULT 1000 CHECK (coins >= 0),
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	
	
	CREATE TABLE IF NOT EXISTS merch (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		price INTEGER NOT NULL CHECK (price > 0)
	);
	INSERT INTO merch (name, price) VALUES
	('t-shirt', 80), ('cup', 20), ('book', 50),
	('pen', 10), ('powerbank', 200), ('hoody', 300),
	('umbrella', 200), ('socks', 10), ('wallet', 50),
	('pink-hoody', 500);
	
	CREATE TABLE IF NOT EXISTS transfers (
		id SERIAL PRIMARY KEY,
		from_user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		to_user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		amount INTEGER NOT NULL CHECK (amount > 0),
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX idx_transfers_from ON transfers(from_user_id);
	CREATE INDEX idx_transfers_to ON transfers(to_user_id);
	
	CREATE TABLE IF NOT EXISTS purchases (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		merch_id INTEGER REFERENCES merch(id) ON DELETE CASCADE,
		quantity INTEGER NOT NULL CHECK (quantity > 0),
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX idx_purchases_user ON purchases(user_id);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	_, err = prepare.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return &Storage{db: db}, nil
}
