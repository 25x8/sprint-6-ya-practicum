package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/middleware"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/models"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
	"github.com/gorilla/mux"
)

// Константы для проверки номера заказа
const (
	CookieName = "auth_token"
)

// Handler представляет обработчики для API
type Handler struct {
	repo          repository.Repository
	auth          *auth.Auth
	accrualClient *accrual.Client
}

// NewHandler создает новый экземпляр Handler
func NewHandler(repo repository.Repository, auth *auth.Auth, accrualClient *accrual.Client) *Handler {
	return &Handler{
		repo:          repo,
		auth:          auth,
		accrualClient: accrualClient,
	}
}

// RegisterRoutes регистрирует маршруты для API
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Создаем middleware для аутентификации
	authMiddleware := middleware.NewAuthMiddleware(h.auth)

	// Регистрируем маршруты без аутентификации
	router.HandleFunc("/api/user/register", h.Register).Methods("POST")
	router.HandleFunc("/api/user/login", h.Login).Methods("POST")

	// Создаем подмаршрутизатор с middleware для аутентификации
	protected := router.PathPrefix("").Subrouter()
	protected.Use(authMiddleware.Middleware)

	// Регистрируем защищенные маршруты
	protected.HandleFunc("/api/user/orders", h.GetOrders).Methods("GET")
	protected.HandleFunc("/api/user/orders", h.CreateOrder).Methods("POST")
	protected.HandleFunc("/api/user/balance", h.GetBalance).Methods("GET")
	protected.HandleFunc("/api/user/balance/withdraw", h.Withdraw).Methods("POST")
	protected.HandleFunc("/api/user/withdrawals", h.GetWithdrawals).Methods("GET")
}

// Register обрабатывает запрос на регистрацию пользователя
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Декодируем запрос
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем логин и пароль
	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}

	// Регистрируем пользователя
	token, err := h.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		log.Printf("Failed to register user: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	// Устанавливаем токен в заголовок Authorization
	w.Header().Set("Authorization", "Bearer "+token)

	// Устанавливаем cookie с токеном
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
}

// Login обрабатывает запрос на аутентификацию пользователя
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Декодируем запрос
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем логин и пароль
	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}

	// Аутентифицируем пользователя
	token, err := h.auth.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		log.Printf("Failed to login user: %v", err)
		http.Error(w, "Failed to login user", http.StatusInternalServerError)
		return
	}

	// Устанавливаем токен в заголовок Authorization
	w.Header().Set("Authorization", "Bearer "+token)

	// Устанавливаем cookie с токеном
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
}

// GetOrders обрабатывает запрос на получение списка заказов пользователя
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем заказы пользователя
	orders, err := h.repo.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to get orders: %v", err)
		http.Error(w, "Failed to get orders", http.StatusInternalServerError)
		return
	}

	// Если заказов нет, возвращаем 204 No Content
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// CreateOrder обрабатывает запрос на создание заказа
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Получаем номер заказа из тела запроса
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	// Проверяем номер заказа по алгоритму Луна
	if !validateLuhn(orderNumber) {
		http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	// Проверяем, существует ли заказ
	order, err := h.repo.GetOrderByNumber(r.Context(), orderNumber)
	if err != nil {
		log.Printf("Failed to check order: %v", err)
		http.Error(w, "Failed to check order", http.StatusInternalServerError)
		return
	}

	// Если заказ существует и принадлежит другому пользователю
	if order != nil && order.UserID != userID {
		http.Error(w, "Order already exists", http.StatusConflict)
		return
	}

	// Если заказ существует и принадлежит текущему пользователю
	if order != nil && order.UserID == userID {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Создаем заказ
	err = h.repo.CreateOrder(r.Context(), userID, orderNumber)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Запускаем горутину для проверки заказа в системе accrual
	go h.checkOrderStatus(orderNumber)

	w.WriteHeader(http.StatusAccepted)
}

// GetBalance обрабатывает запрос на получение баланса пользователя
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем баланс пользователя
	balance, err := h.repo.GetUserBalance(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// Withdraw обрабатывает запрос на списание баллов
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодируем запрос
	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем номер заказа и сумму
	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "Order number and sum are required", http.StatusBadRequest)
		return
	}

	// Проверяем номер заказа по алгоритму Луна
	if !validateLuhn(req.Order) {
		http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
		return
	}

	// Списываем баллы
	err := h.repo.CreateWithdrawal(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
			return
		}
		log.Printf("Failed to withdraw: %v", err)
		http.Error(w, "Failed to withdraw", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetWithdrawals обрабатывает запрос на получение списка операций списания
func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем список операций списания
	withdrawals, err := h.repo.GetWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Failed to get withdrawals: %v", err)
		http.Error(w, "Failed to get withdrawals", http.StatusInternalServerError)
		return
	}

	// Если операций нет, возвращаем 204 No Content
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}

// checkOrderStatus проверяет статус заказа в системе accrual
func (h *Handler) checkOrderStatus(orderNumber string) {
	// Создаем контекст с таймаутом
	ctx := context.Background()

	// Периодически проверяем статус заказа
	for {
		// Получаем информацию о заказе из системы accrual
		orderInfo, err := h.accrualClient.GetOrder(orderNumber)
		if err != nil {
			// Если ошибка связана с ограничением запросов, ждем указанное время
			if strings.Contains(err.Error(), "too many requests") {
				// Извлекаем время ожидания из сообщения об ошибке
				var waitTime int
				_, err := fmt.Sscanf(err.Error(), "too many requests, retry after %d seconds", &waitTime)
				if err != nil || waitTime <= 0 {
					waitTime = 60 // По умолчанию ждем 60 секунд
				}
				log.Printf("Rate limit exceeded, waiting %d seconds before retry", waitTime)
				time.Sleep(time.Duration(waitTime) * time.Second)
				continue
			}

			log.Printf("Failed to get order info: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}

		// Если заказ не найден, продолжаем проверку
		if orderInfo == nil {
			time.Sleep(time.Second * 5)
			continue
		}

		// Обновляем статус заказа
		err = h.repo.UpdateOrderStatus(ctx, orderNumber, orderInfo.Status, orderInfo.Accrual)
		if err != nil {
			log.Printf("Failed to update order status: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}

		// Если статус PROCESSED, начисляем баллы пользователю
		if orderInfo.Status == models.StatusProcessed {
			// Получаем заказ из базы данных
			order, err := h.repo.GetOrderByNumber(ctx, orderNumber)
			if err != nil {
				log.Printf("Failed to get order: %v", err)
				break
			}

			// Начисляем баллы пользователю
			err = h.repo.UpdateUserBalance(ctx, order.UserID, orderInfo.Accrual, 0)
			if err != nil {
				log.Printf("Failed to update user balance: %v", err)
				break
			}

			// Завершаем проверку
			break
		}

		// Если статус INVALID, завершаем проверку
		if orderInfo.Status == models.StatusInvalid {
			break
		}

		// Ждем перед следующей проверкой
		time.Sleep(time.Second * 5)
	}
}

// validateLuhn проверяет номер заказа по алгоритму Луна
func validateLuhn(number string) bool {
	// Проверяем, что номер состоит только из цифр
	for _, r := range number {
		if r < '0' || r > '9' {
			return false
		}
	}

	// Алгоритм Луна
	sum := 0
	numDigits := len(number)
	parity := numDigits % 2

	for i := 0; i < numDigits; i++ {
		digit := int(number[i] - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return sum%10 == 0
}
