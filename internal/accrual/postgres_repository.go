package accrual

import (
	"context"
	"database/sql"
	"errors"
)

// PostgresRepository реализует интерфейс Repository для PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateOrder создает новый заказ в базе данных
func (r *PostgresRepository) CreateOrder(ctx context.Context, number string, status OrderStatus, accrual float64) error {
	query := `
		INSERT INTO orders (number, status, accrual)
		VALUES ($1, $2, $3)
		ON CONFLICT (number) DO UPDATE
		SET status = EXCLUDED.status, accrual = EXCLUDED.accrual
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, number, status, accrual).Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

// GetOrder возвращает информацию о заказе
func (r *PostgresRepository) GetOrder(ctx context.Context, number string) (*Order, error) {
	query := `
		SELECT number, status, accrual
		FROM orders
		WHERE number = $1`

	order := &Order{}
	err := r.db.QueryRowContext(ctx, query, number).Scan(&order.Number, &order.Status, &order.Accrual)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return order, nil
}

// UpdateOrderStatus обновляет статус заказа
func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, number string, status OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $2
		WHERE number = $1`

	result, err := r.db.ExecContext(ctx, query, number, status)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateOrderAccrual обновляет начисление для заказа
func (r *PostgresRepository) UpdateOrderAccrual(ctx context.Context, number string, accrual float64) error {
	query := `
		UPDATE orders
		SET accrual = $2
		WHERE number = $1`

	result, err := r.db.ExecContext(ctx, query, number, accrual)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// SaveMechanic сохраняет механику начисления баллов в базе данных
func (r *PostgresRepository) SaveMechanic(ctx context.Context, mechanic *Mechanic) error {
	query := `
		INSERT INTO mechanics (match, reward, reward_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (match) DO UPDATE
		SET reward = EXCLUDED.reward, reward_type = EXCLUDED.reward_type
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, mechanic.Match, mechanic.Reward, mechanic.RewardType).Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

// GetMechanics возвращает список всех механик начисления баллов
func (r *PostgresRepository) GetMechanics(ctx context.Context) ([]*Mechanic, error) {
	query := `
		SELECT match, reward, reward_type
		FROM mechanics`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mechanics := make([]*Mechanic, 0)
	for rows.Next() {
		mechanic := &Mechanic{}
		err := rows.Scan(&mechanic.Match, &mechanic.Reward, &mechanic.RewardType)
		if err != nil {
			return nil, err
		}
		mechanics = append(mechanics, mechanic)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mechanics, nil
}
