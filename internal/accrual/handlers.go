package accrual

import (
	"context"
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

// Временные интервалы для изменения статуса заказа
const (
	ProcessingDelay = 5 * time.Second  // Время перехода в статус PROCESSING
	ProcessedDelay  = 10 * time.Second // Время перехода в статус PROCESSED

	// Максимальное количество одновременно обрабатываемых заказов
	MaxConcurrentOrders = 20
)

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

// orderTask представляет задачу на обработку заказа
type orderTask struct {
	orderNumber string
}

type Handler struct {
	repo        Repository
	rateLimiter *rateLimiter
	orderTasks  chan orderTask
	workerPool  sync.WaitGroup
}

func NewHandler(repo Repository) *Handler {
	h := &Handler{
		repo:        repo,
		rateLimiter: newRateLimiter(),
		orderTasks:  make(chan orderTask, MaxConcurrentOrders),
	}

	// Запускаем воркеры для обработки заказов
	h.startWorkers()

	return h
}

// startWorkers запускает пул воркеров для обработки заказов
func (h *Handler) startWorkers() {
	for i := 0; i < MaxConcurrentOrders; i++ {
		h.workerPool.Add(1)
		go func() {
			defer h.workerPool.Done()
			for task := range h.orderTasks {
				h.processOrder(task.orderNumber)
			}
		}()
	}
}

// Shutdown останавливает обработку заказов
func (h *Handler) Shutdown() {
	close(h.orderTasks)
	h.workerPool.Wait()
}

// validateLuhn проверяет, соответствует ли номер заказа алгоритму Луна
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

	// Проверяем корректность номера заказа по алгоритму Луна
	if !validateLuhn(orderNumber) {
		http.Error(w, "Invalid order number format", http.StatusBadRequest)
		return
	}

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

	// Проверяем корректность номера заказа по алгоритму Луна
	if !validateLuhn(request.Order) {
		http.Error(w, "Invalid order number format", http.StatusBadRequest)
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

	// Отправляем задачу на обработку в пул воркеров
	select {
	case h.orderTasks <- orderTask{
		orderNumber: order.Number,
	}:
		// Задача успешно отправлена в очередь
	default:
		// Очередь заполнена, но мы все равно принимаем заказ
		// и запускаем обработку в отдельной горутине
		go h.processOrderAsync(order.Number)
	}

	w.WriteHeader(http.StatusAccepted)
}

// processOrder обрабатывает заказ (используется воркерами)
func (h *Handler) processOrder(orderNumber string) {
	// Через 5 секунд меняем статус на PROCESSING
	time.Sleep(ProcessingDelay)

	// Создаем контекст для работы с базой данных
	ctx := context.Background()

	err := h.repo.UpdateOrderStatus(ctx, orderNumber, "PROCESSING")
	if err != nil {
		log.Printf("Failed to update order status to PROCESSING: %v", err)
		return
	}

	// Через 10 секунд меняем статус на PROCESSED и рассчитываем начисление
	time.Sleep(ProcessedDelay)

	// Рассчитываем начисление с помощью базы данных
	totalAccrual, err := h.repo.CalculateOrderAccrual(ctx, orderNumber)
	if err != nil {
		log.Printf("Failed to calculate order accrual: %v", err)
		return
	}

	// Обновляем статус заказа и начисление
	err = h.repo.UpdateOrderStatusAndAccrual(ctx, orderNumber, "PROCESSED", totalAccrual)
	if err != nil {
		log.Printf("Failed to update order status to PROCESSED: %v", err)
		return
	}

	log.Printf("Order %s processed successfully with accrual: %.2f", orderNumber, totalAccrual)
}

// processOrderAsync обрабатывает заказ асинхронно (используется при переполнении очереди)
func (h *Handler) processOrderAsync(orderNumber string) {
	h.processOrder(orderNumber)
}

func (h *Handler) AddGoodsReward(w http.ResponseWriter, r *http.Request) {
	var reward GoodsReward

	if err := json.NewDecoder(r.Body).Decode(&reward); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем тип вознаграждения
	if reward.RewardType != "pt" && reward.RewardType != "%" {
		log.Printf("Invalid reward type: %s", reward.RewardType)
		http.Error(w, "Invalid reward type. Must be '%' or 'pt'", http.StatusBadRequest)
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
