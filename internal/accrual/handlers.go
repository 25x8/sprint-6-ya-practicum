package accrual

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Handler представляет обработчик HTTP-запросов
type Handler struct {
	repo      Repository
	mechanics []*Mechanic
	mu        sync.RWMutex
	workers   *sync.WaitGroup
}

// NewHandler создает новый экземпляр Handler
func NewHandler(repo Repository) *Handler {
	h := &Handler{
		repo:      repo,
		mechanics: make([]*Mechanic, 0),
		workers:   &sync.WaitGroup{},
	}
	return h
}

// RegisterRoutes регистрирует маршруты для API
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/goods", h.RegisterMechanic).Methods(http.MethodPost)
	router.HandleFunc("/api/orders", h.RegisterOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/orders/{number}", h.GetOrder).Methods(http.MethodGet)
}

// RegisterMechanic регистрирует новое правило начисления баллов
func (h *Handler) RegisterMechanic(w http.ResponseWriter, r *http.Request) {
	var mechanic Mechanic
	if err := json.NewDecoder(r.Body).Decode(&mechanic); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Сохраняем в базе данных
	if err := h.repo.SaveMechanic(r.Context(), &mechanic); err != nil {
		log.Printf("Failed to save mechanic: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Добавляем в кэш
	h.mu.Lock()
	h.mechanics = append(h.mechanics, &mechanic)
	h.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// RegisterOrder регистрирует новый заказ
func (h *Handler) RegisterOrder(w http.ResponseWriter, r *http.Request) {
	var orderReq OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Рассчитываем начисление для заказа
	var totalAccrual float64
	h.mu.RLock()
	for _, good := range orderReq.Goods {
		for _, mechanic := range h.mechanics {
			if contains(good.Description, mechanic.Match) {
				if mechanic.RewardType == "%" {
					totalAccrual += good.Price * mechanic.Reward / 100
				} else {
					totalAccrual += mechanic.Reward
				}
			}
		}
	}
	h.mu.RUnlock()

	// Создаем заказ со статусом NEW
	if err := h.repo.CreateOrder(r.Context(), orderReq.Order, StatusNew, 0); err != nil {
		log.Printf("Failed to create order: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Запускаем асинхронную обработку заказа
	h.workers.Add(1)
	go h.processOrder(orderReq.Order, totalAccrual)

	w.WriteHeader(http.StatusAccepted)
}

// GetOrder возвращает информацию о заказе
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	number := vars["number"]

	order, err := h.repo.GetOrder(r.Context(), number)
	if err != nil {
		log.Printf("Failed to get order: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if order == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// processOrder обрабатывает заказ асинхронно
func (h *Handler) processOrder(orderNumber string, accrual float64) {
	defer h.workers.Done()

	// Сначала меняем статус на PROCESSING
	ctx := context.Background()
	if err := h.repo.UpdateOrderStatus(ctx, orderNumber, StatusProcessing); err != nil {
		log.Printf("Failed to update order status to PROCESSING: %v", err)
		return
	}

	// Небольшая задержка для имитации обработки
	time.Sleep(2 * time.Second)

	// Затем устанавливаем статус PROCESSED и начисление
	if err := h.repo.UpdateOrderStatus(ctx, orderNumber, StatusProcessed); err != nil {
		log.Printf("Failed to update order status to PROCESSED: %v", err)
		return
	}

	if err := h.repo.UpdateOrderAccrual(ctx, orderNumber, accrual); err != nil {
		log.Printf("Failed to update order accrual: %v", err)
		return
	}

	log.Printf("Order %s processed with accrual %.2f", orderNumber, accrual)
}

// Shutdown корректно завершает работу обработчика
func (h *Handler) Shutdown() {
	h.workers.Wait()
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && s != substr &&
		hasSubstring(s, substr)
}

// hasSubstring проверяет, входит ли подстрока в строку
func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
