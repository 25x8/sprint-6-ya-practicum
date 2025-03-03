package accrual

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderNumber := vars["number"]

	order, err := h.repo.GetOrder(r.Context(), orderNumber)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	order := &Order{
		Number:  request.Order,
		Status:  "REGISTERED",
		Accrual: 0,
	}

	err := h.repo.CreateOrder(r.Context(), order, request.Goods)
	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) AddGoodsReward(w http.ResponseWriter, r *http.Request) {
	var reward GoodsReward

	if err := json.NewDecoder(r.Body).Decode(&reward); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.repo.AddGoodsReward(r.Context(), &reward)
	if err != nil {
		http.Error(w, "Failed to add goods reward", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/orders/{number}", h.GetOrder).Methods("GET")
	router.HandleFunc("/api/orders", h.CreateOrderHandler).Methods("POST")
	router.HandleFunc("/api/goods/rewards", h.AddGoodsReward).Methods("POST")
}
