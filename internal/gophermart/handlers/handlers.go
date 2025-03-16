package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/accrual"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/models"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
	"github.com/gorilla/mux"
)

// Константы для проверки номера заказа
const (
	CookieName = "auth_token"
)

// Константа для ключа userID в контексте
const userIDKey = "userID"

// Handler представляет обработчики для API
type Handler struct {
	repo          repository.Repository
	auth          *auth.Auth
	accrualClient *accrual.Client
	logger        *slog.Logger
}

// NewHandler создает новый экземпляр Handler
func NewHandler(repo repository.Repository, auth *auth.Auth, accrualClient *accrual.Client, logger *slog.Logger) *Handler {
	return &Handler{
		repo:          repo,
		auth:          auth,
		accrualClient: accrualClient,
		logger:        logger,
	}
}

// RegisterRoutes регистрирует маршруты для API
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Регистрируем маршруты без аутентификации
	router.HandleFunc("/api/user/register", h.Register).Methods("POST")
	router.HandleFunc("/api/user/login", h.Login).Methods("POST")

	// Создаем подмаршрутизатор с middleware для аутентификации
	protected := router.PathPrefix("").Subrouter()
	protected.Use(h.auth.AuthMiddleware)

	// Регистрируем защищенные маршруты
	protected.HandleFunc("/api/user/orders", h.GetOrders).Methods("GET")
	protected.HandleFunc("/api/user/orders", h.CreateOrder).Methods("POST")
	protected.HandleFunc("/api/user/balance", h.GetBalance).Methods("GET")
	protected.HandleFunc("/api/user/balance/add", h.AddBalance).Methods("POST")
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
	userID, ok := auth.GetUserIDFromContext(r.Context())
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
	userID, ok := auth.GetUserIDFromContext(r.Context())
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

	// Отправляем ответ
	w.WriteHeader(http.StatusAccepted)
}

// validateLuhn проверяет номер заказа по алгоритму Луна
func validateLuhn(number string) bool {
	// Проверяем, что номер содержит только цифры
	for _, r := range number {
		if r < '0' || r > '9' {
			return false
		}
	}

	// Реализация алгоритма Луна
	sum := 0
	parity := len(number) % 2
	for i, digit := range number {
		n, _ := strconv.Atoi(string(digit))
		if i%2 == parity {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}
