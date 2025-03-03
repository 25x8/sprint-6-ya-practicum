CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('REGISTERED', 'INVALID', 'PROCESSING', 'PROCESSED')),
    accrual DECIMAL(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)

CREATE TABLE order_goods (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    price DECIMAL(10, 2) NOT NULL
)

CREATE TABLE goods_rewards (
    id SERIAL PRIMARY KEY,
    match TEXT NOT NULL,
    reward DECIMAL(10, 2) NOT NULL,
    reward_type VARCHAR(10) NOT NULL CHECK (reward_type IN ('%', 'pt'))
)