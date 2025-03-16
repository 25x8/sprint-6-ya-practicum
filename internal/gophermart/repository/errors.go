package repository

import "errors"

// Определения ошибок
var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrOrderExists        = errors.New("order already exists")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
