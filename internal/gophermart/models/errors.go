package models

import "errors"

// Ошибки бизнес-логики
var (
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrOrderAlreadyExists        = errors.New("order already exists")
	ErrOrderBelongsToAnotherUser = errors.New("order belongs to another user")
	ErrInsufficientFunds         = errors.New("insufficient funds")
)
