package accrual

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// RequestLimit определяет максимальное количество запросов в минуту
const RequestLimit = 10

// rateLimiter хранит информацию о запросах для ограничения их количества
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string][]time.Time
}

// newRateLimiter создает новый экземпляр ограничителя запросов
func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		clients: make(map[string][]time.Time),
	}
}

// isAllowed проверяет, не превышен ли лимит запросов для клиента
func (rl *rateLimiter) isAllowed(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	minuteAgo := now.Add(-time.Minute)

	// Получаем текущие записи для клиента или создаем новый слайс
	times, exists := rl.clients[clientIP]
	newTimes := []time.Time{}

	// Фильтруем только актуальные записи (не старше минуты)
	if exists {
		for _, t := range times {
			if t.After(minuteAgo) {
				newTimes = append(newTimes, t)
			}
		}
	}

	// Проверяем лимит до добавления нового запроса
	if len(newTimes) >= RequestLimit {
		return false
	}

	// Добавляем новый запрос и обновляем записи
	newTimes = append(newTimes, now)
	rl.clients[clientIP] = newTimes

	return true
}

type Handler struct {
	repo        Repository
	rateLimiter *rateLimiter
}

func NewHandler(repo Repository) *Handler {
	return &Handler{
		repo:        repo,
		rateLimiter: newRateLimiter(),
	}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Получаем IP-адрес клиента
	clientIP := r.RemoteAddr

	// Проверяем лимит запросов
	if !h.rateLimiter.isAllowed(clientIP) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Retry-After", "60")
		http.Error(w, fmt.Sprintf("No more than %d requests per minute allowed", RequestLimit), http.StatusTooManyRequests)
		return
	}

	vars := mux.Vars(r)
	orderNumber := vars["number"]

	order, err := h.repo.GetOrder(r.Context(), orderNumber)

	if err != nil {
		log.Printf("Failed to get order: %v", err)
		http.Error(w, "Failed to get order", http.StatusInternalServerError)
		return
	}

	if order == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Order string       `json:"order"`
		Goods []OrderGoods `json:"goods"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.repo.GetOrder(r.Context(), request.Order)
	if err == nil && order != nil {
		log.Printf("Order already exists")
		http.Error(w, "Order already exists", http.StatusConflict)
		return
	}

	order = &Order{
		Number:  request.Order,
		Status:  "REGISTERED",
		Accrual: 0,
	}

	err = h.repo.CreateOrder(r.Context(), order, request.Goods)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) AddGoodsReward(w http.ResponseWriter, r *http.Request) {
	var reward GoodsReward

	if err := json.NewDecoder(r.Body).Decode(&reward); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reward.RewardType != "%" && reward.RewardType != "pt" {
		log.Printf("Invalid reward type")
		http.Error(w, "Invalid reward type", http.StatusBadRequest)
		return
	}

	existingReward, err := h.repo.GetGoodsReward(r.Context(), reward.Match)
	if err == nil && existingReward != nil {
		log.Printf("Goods reward already exists")
		http.Error(w, "Goods reward already exists", http.StatusConflict)
		return
	}

	err = h.repo.AddGoodsReward(r.Context(), &reward)
	if err != nil {
		log.Printf("Failed to add goods reward: %v", err)
		http.Error(w, "Failed to add goods reward", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/orders/{number}", h.GetOrder).Methods("GET")
	router.HandleFunc("/api/orders", h.CreateOrderHandler).Methods("POST")
	router.HandleFunc("/api/goods", h.AddGoodsReward).Methods("POST")
}
