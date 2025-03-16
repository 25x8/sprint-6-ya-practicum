package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/models"
)

// Repository определяет интерфейс для работы с базой данных
type Repository interface {
	// Методы для работы с пользователями
	CreateUser(ctx context.Context, login, passwordHash string) (int, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	UpdateUserBalance(ctx context.Context, userID int, balanceDelta float64, withdrawnDelta float64) error

	// Методы для работы с заказами
	CreateOrder(ctx context.Context, userID int, orderNumber string) error
	GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderNumber, status string, accrual float64) error

	// Методы для работы с выводом средств
	CreateWithdrawal(ctx context.Context, userID int, orderNumber string, sum float64) error
	GetWithdrawalsByUserID(ctx context.Context, userID int) ([]models.Withdrawal, error)

	// Метод для проверки баланса пользователя
	GetUserBalance(ctx context.Context, userID int) (*models.Balance, error)

	// Метод для добавления баланса пользователю
	AddBalanceToUser(ctx context.Context, userID int, amount float64) error
}

// PostgresRepository реализует интерфейс Repository для PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateUser создает нового пользователя
func (r *PostgresRepository) CreateUser(ctx context.Context, login, passwordHash string) (int, error) {
	var id int
	query := `INSERT INTO users (login, password_hash, balance, withdrawn, created_at) 
			  VALUES ($1, $2, 0, 0, $3) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, login, passwordHash, time.Now()).Scan(&id)
	return id, err
}

// GetUserByLogin возвращает пользователя по логину
func (r *PostgresRepository) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User
	query := `SELECT id, login, password_hash, balance, withdrawn, created_at FROM users WHERE login = $1`
	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.Balance, &user.Withdrawn, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID возвращает пользователя по ID
func (r *PostgresRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User
	query := `SELECT id, login, password_hash, balance, withdrawn, created_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.Balance, &user.Withdrawn, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUserBalance обновляет баланс пользователя
func (r *PostgresRepository) UpdateUserBalance(ctx context.Context, userID int, balanceDelta float64, withdrawnDelta float64) error {
	query := `UPDATE users SET balance = balance + $1, withdrawn = withdrawn + $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, balanceDelta, withdrawnDelta, userID)
	return err
}

// CreateOrder создает новый заказ
func (r *PostgresRepository) CreateOrder(ctx context.Context, userID int, orderNumber string) error {
	query := `INSERT INTO orders (user_id, number, status, accrual, uploaded_at) 
			  VALUES ($1, $2, $3, 0, $4)`
	_, err := r.db.ExecContext(ctx, query, userID, orderNumber, models.StatusNew, time.Now())
	return err
}

// GetOrderByNumber возвращает заказ по номеру
func (r *PostgresRepository) GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	var order models.Order
	query := `SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE number = $1`
	err := r.db.QueryRowContext(ctx, query, orderNumber).Scan(
		&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetOrdersByUserID возвращает все заказы пользователя
func (r *PostgresRepository) GetOrdersByUserID(ctx context.Context, userID int) ([]models.Order, error) {
	query := `SELECT id, user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// UpdateOrderStatus обновляет статус заказа
func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, orderNumber, status string, accrual float64) error {
	query := `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`
	_, err := r.db.ExecContext(ctx, query, status, accrual, orderNumber)
	return err
}

// CreateWithdrawal создает новую операцию вывода средств
func (r *PostgresRepository) CreateWithdrawal(ctx context.Context, userID int, orderNumber string, sum float64) error {
	fmt.Printf("CreateWithdrawal: userID=%d, orderNumber=%s, sum=%f\n", userID, orderNumber, sum)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("Error beginning transaction: %v\n", err)
		return err
	}

	// Проверяем достаточно ли средств
	var balance float64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance)
	if err != nil {
		tx.Rollback()
		fmt.Printf("Error getting balance: %v\n", err)
		return err
	}

	fmt.Printf("User balance: %f, withdrawal sum: %f\n", balance, sum)

	if balance < sum {
		tx.Rollback()
		fmt.Printf("Insufficient funds: balance=%f, sum=%f\n", balance, sum)
		return ErrInsufficientFunds
	}

	// Обновляем баланс пользователя
	_, err = tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, withdrawn = withdrawn + $1 WHERE id = $2`, sum, userID)
	if err != nil {
		tx.Rollback()
		fmt.Printf("Error updating balance: %v\n", err)
		return err
	}

	// Создаем запись о выводе средств
	_, err = tx.ExecContext(ctx, `INSERT INTO withdrawals (user_id, order_number, sum, processed_at) VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, sum, time.Now())
	if err != nil {
		tx.Rollback()
		fmt.Printf("Error inserting withdrawal: %v\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		return err
	}

	fmt.Printf("Withdrawal created successfully\n")
	return nil
}

// GetWithdrawalsByUserID возвращает все операции вывода средств пользователя
func (r *PostgresRepository) GetWithdrawalsByUserID(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	query := `SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var withdrawal models.Withdrawal
		if err := rows.Scan(&withdrawal.ID, &withdrawal.UserID, &withdrawal.OrderNumber, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return withdrawals, nil
}

// GetUserBalance возвращает баланс пользователя
func (r *PostgresRepository) GetUserBalance(ctx context.Context, userID int) (*models.Balance, error) {
	var balance models.Balance
	query := `SELECT balance, withdrawn FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

// AddBalanceToUser добавляет указанную сумму к балансу пользователя
func (r *PostgresRepository) AddBalanceToUser(ctx context.Context, userID int, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	query := `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, amount, userID)
	if err != nil {
		return err
	}

	return nil
}
