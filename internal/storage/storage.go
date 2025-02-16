package storage

import "errors"

var (
	ErrNotEnoughCoins    = errors.New("not enough coins")
	ErrUserNotFound      = errors.New("user not found")
	ErrItemNotFound      = errors.New("item not found")
	ErrPasswordUnmatched = errors.New("password unmatched")
)
