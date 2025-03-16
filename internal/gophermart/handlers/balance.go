package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/models"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
)

// GetBalance возвращает текущий баланс пользователя
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем баланс пользователя
	balance, err := h.repo.GetUserBalance(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user balance", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// Withdraw обрабатывает запрос на списание средств с баланса пользователя
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var withdrawalRequest models.WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&withdrawalRequest); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding request: %v\n", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(os.Stderr, "Withdrawal request: %+v\n", withdrawalRequest)

	// Проверяем номер заказа на соответствие алгоритму Луна
	if !validateLuhn(withdrawalRequest.Order) {
		fmt.Fprintf(os.Stderr, "Invalid order number format: %s\n", withdrawalRequest.Order)
		http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	// Списываем средства
	err := h.repo.CreateWithdrawal(r.Context(), userID, withdrawalRequest.Order, withdrawalRequest.Sum)
	if err == repository.ErrInsufficientFunds {
		fmt.Fprintf(os.Stderr, "Insufficient funds for user %d\n", userID)
		http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating withdrawal: %v\n", err)
		h.logger.Error("Failed to create withdrawal", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals возвращает список списаний пользователя
func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем список списаний пользователя
	withdrawals, err := h.repo.GetWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user withdrawals", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Если списаний нет, возвращаем пустой список
	if len(withdrawals) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}

// AddBalance обрабатывает запрос на пополнение баланса пользователя
func (h *Handler) AddBalance(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var addBalanceRequest struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&addBalanceRequest); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding add balance request: %v\n", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Проверяем сумму
	if addBalanceRequest.Amount <= 0 {
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	// Пополняем баланс
	err := h.repo.AddBalanceToUser(r.Context(), userID, addBalanceRequest.Amount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding balance: %v\n", err)
		h.logger.Error("Failed to add balance", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Получаем обновленный баланс
	balance, err := h.repo.GetUserBalance(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get updated balance", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}
