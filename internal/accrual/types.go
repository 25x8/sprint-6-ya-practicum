package accrual

import "context"

// OrderStatus представляет статус обработки заказа
type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
)

// Order представляет информацию о заказе
type Order struct {
	Number  string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual,omitempty"`
}

// OrderRequest представляет запрос на регистрацию заказа
type OrderRequest struct {
	Order string      `json:"order"`
	Goods []GoodsItem `json:"goods"`
}

// GoodsItem представляет информацию о товаре
type GoodsItem struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// Mechanic представляет правило начисления баллов
type Mechanic struct {
	Match      string  `json:"match"`
	Reward     float64 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

// Repository определяет интерфейс для работы с хранилищем данных
type Repository interface {
	CreateOrder(ctx context.Context, number string, status OrderStatus, accrual float64) error
	GetOrder(ctx context.Context, number string) (*Order, error)
	UpdateOrderStatus(ctx context.Context, number string, status OrderStatus) error
	UpdateOrderAccrual(ctx context.Context, number string, accrual float64) error
	SaveMechanic(ctx context.Context, mechanic *Mechanic) error
	GetMechanics(ctx context.Context) ([]*Mechanic, error)
}
