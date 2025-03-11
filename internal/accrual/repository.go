package accrual

import (
	"context"
	"database/sql"
)

type Order struct {
	Number  string  `json:"number"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type OrderGoods struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type GoodsReward struct {
	Match      string  `json:"match"`
	Reward     float64 `json:"reward"`
	RewardType string  `json:"reward_type"` // "%" или "pt"
}

type Repository interface {
	GetOrder(ctx context.Context, orderNumber string) (*Order, error)
	GetGoodsReward(ctx context.Context, match string) (*GoodsReward, error)
	CreateOrder(ctx context.Context, order *Order, goods []OrderGoods) error
	AddGoodsReward(ctx context.Context, reward *GoodsReward) error
	UpdateOrderStatus(ctx context.Context, orderNumber string, status string) error
	UpdateOrderStatusAndAccrual(ctx context.Context, orderNumber string, status string, accrual float64) error
	GetAllRewards(ctx context.Context) ([]GoodsReward, error)
	CalculateOrderAccrual(ctx context.Context, orderNumber string) (float64, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderNumber string) (*Order, error) {
	var order Order

	query := `SELECT order_number, status, accrual FROM orders WHERE order_number = $1`
	err := r.db.QueryRowContext(ctx, query, orderNumber).Scan(&order.Number, &order.Status, &order.Accrual)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *PostgresRepository) CreateOrder(ctx context.Context, order *Order, goods []OrderGoods) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	query := `INSERT INTO orders (order_number, status, accrual) VALUES ($1, $2, $3) RETURNING id`
	var orderID int

	err = tx.QueryRowContext(ctx, query, &order.Number, &order.Status, &order.Accrual).Scan(&orderID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, g := range goods {
		_, err := tx.ExecContext(ctx, `INSERT INTO order_goods (order_id, description, price) VALUES ($1, $2, $3)`, orderID, g.Description, g.Price)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) AddGoodsReward(ctx context.Context, reward *GoodsReward) error {
	query := `INSERT INTO goods_rewards (match, reward, reward_type) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, reward.Match, reward.Reward, reward.RewardType)
	return err
}

func (r *PostgresRepository) GetGoodsReward(ctx context.Context, match string) (*GoodsReward, error) {
	query := `SELECT match, reward, reward_type FROM goods_rewards WHERE match = $1`
	var reward GoodsReward
	err := r.db.QueryRowContext(ctx, query, match).Scan(&reward.Match, &reward.Reward, &reward.RewardType)
	return &reward, err
}

func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, orderNumber string, status string) error {
	query := `UPDATE orders SET status = $1 WHERE order_number = $2`
	_, err := r.db.ExecContext(ctx, query, status, orderNumber)
	return err
}

func (r *PostgresRepository) UpdateOrderStatusAndAccrual(ctx context.Context, orderNumber string, status string, accrual float64) error {
	query := `UPDATE orders SET status = $1, accrual = $2 WHERE order_number = $3`
	_, err := r.db.ExecContext(ctx, query, status, accrual, orderNumber)
	return err
}

func (r *PostgresRepository) GetAllRewards(ctx context.Context) ([]GoodsReward, error) {
	query := `SELECT match, reward, reward_type FROM goods_rewards`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rewards []GoodsReward
	for rows.Next() {
		var reward GoodsReward
		if err := rows.Scan(&reward.Match, &reward.Reward, &reward.RewardType); err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rewards, nil
}

// CalculateOrderAccrual вычисляет начисление для заказа на основе совпадений товаров с вознаграждениями
func (r *PostgresRepository) CalculateOrderAccrual(ctx context.Context, orderNumber string) (float64, error) {
	// Запрос для расчета начисления на основе совпадений товаров с вознаграждениями
	query := `
		WITH order_items AS (
			SELECT og.description, og.price
			FROM order_goods og
			JOIN orders o ON og.order_id = o.id
			WHERE o.order_number = $1
		)
		SELECT 
			COALESCE(SUM(
				CASE 
					WHEN gr.reward_type = 'pt' THEN gr.reward
					WHEN gr.reward_type = '%' THEN (oi.price * gr.reward) / 100
					ELSE 0
				END
			), 0) as total_accrual
		FROM order_items oi
		CROSS JOIN goods_rewards gr
		WHERE LOWER(oi.description) LIKE '%' || LOWER(gr.match) || '%'
	`

	var totalAccrual float64
	err := r.db.QueryRowContext(ctx, query, orderNumber).Scan(&totalAccrual)
	if err != nil {
		return 0, err
	}

	return totalAccrual, nil
}
