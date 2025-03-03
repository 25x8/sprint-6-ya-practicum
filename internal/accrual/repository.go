package accrual

import (
	"context"
	"database/sql"
	"errors"
)

type Order struct {
	Number  string `json:"number"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

type OrderGoods struct {
	Description string `json:"description"`
	Price       string `json:"price"`
}

type GoodsReward struct {
	Match      string `json:"match"`
	Reward     int    `json:"reward"`
	RewardType string `json:"reward_type"` // "%" или "pt"
}

type Repository interface {
	GetOrder(ctx context.Context, orderNumber string) (*Order, error)
	CreateOrder(ctx context.Context, order *Order, goods []OrderGoods) error
	AddGoodsReward(ctx context.Context, reward *GoodsReward) error
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
			return nil, errors.New("order not found")
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
