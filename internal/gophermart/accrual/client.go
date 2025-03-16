package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client представляет клиент для работы с API системы начисления баллов
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// OrderStatus представляет статус обработки заказа
type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusInvalid    OrderStatus = "INVALID"
	StatusProcessed  OrderStatus = "PROCESSED"
)

// Order представляет информацию о заказе, полученную от системы начисления баллов
type Order struct {
	Number  string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual,omitempty"`
}

// NewClient создает новый экземпляр клиента
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetOrder получает информацию о заказе от системы начисления баллов
func (c *Client) GetOrder(orderNumber string) (*Order, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get order from accrual system: %w", err)
	}
	defer resp.Body.Close()

	// Если заказ не найден или еще не обработан
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	// Если произошла ошибка на сервере
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("accrual system returned status code %d", resp.StatusCode)
	}

	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &order, nil
}
