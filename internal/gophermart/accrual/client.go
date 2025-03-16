package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OrderResponse представляет ответ от системы accrual
type OrderResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

// Client представляет клиент для взаимодействия с системой accrual
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создает новый экземпляр Client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetOrder получает информацию о заказе из системы accrual
func (c *Client) GetOrder(orderNumber string) (*OrderResponse, error) {
	// Формируем URL
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	// Отправляем запрос
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	// Если статус 429 (Too Many Requests), возвращаем ошибку с информацией о необходимости повторить запрос позже
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("too many requests, retry after %s seconds", retryAfter)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Декодируем ответ
	var order OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, err
	}

	return &order, nil
}
