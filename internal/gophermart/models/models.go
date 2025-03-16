package models

import (
	"time"
)

// User представляет пользователя системы
type User struct {
	ID           int       `json:"-"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	Balance      float64   `json:"current"`
	Withdrawn    float64   `json:"withdrawn"`
	CreatedAt    time.Time `json:"-"`
}

// Order представляет заказ в системе
type Order struct {
	ID         int       `json:"-"`
	UserID     int       `json:"-"`
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// Withdrawal представляет операцию списания баллов
type Withdrawal struct {
	ID          int       `json:"-"`
	UserID      int       `json:"-"`
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

// Balance представляет баланс пользователя
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// OrderStatus содержит константы для статусов заказа
const (
	StatusNew        = "NEW"        // заказ загружен в систему, но не попал в обработку
	StatusProcessing = "PROCESSING" // заказ в обработке
	StatusInvalid    = "INVALID"    // заказ не принят к расчёту, и вознаграждение не будет начислено
	StatusProcessed  = "PROCESSED"  // расчёт начисления произведён
)

// AuthRequest представляет запрос на аутентификацию/регистрацию
type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// OrderRequest представляет запрос на добавление заказа
type OrderRequest struct {
	Number string `json:"order"`
}

// WithdrawRequest представляет запрос на списание баллов
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
