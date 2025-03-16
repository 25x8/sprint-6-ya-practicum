package models

// UserBalance представляет информацию о балансе пользователя
type UserBalance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// WithdrawalRequest представляет запрос на списание средств с баланса
type WithdrawalRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
