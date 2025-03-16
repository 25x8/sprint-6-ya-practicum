CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    number VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('NEW', 'REGISTERED', 'INVALID', 'PROCESSING', 'PROCESSED')),
    accrual NUMERIC(10, 2),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_goods (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    description TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS goods_rewards (
    id SERIAL PRIMARY KEY,
    match_pattern TEXT NOT NULL,
    reward_type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mechanics (
    id SERIAL PRIMARY KEY,
    match TEXT NOT NULL UNIQUE,
    reward NUMERIC(10, 2) NOT NULL,
    reward_type VARCHAR(10) NOT NULL CHECK (reward_type IN ('%', 'pt')),
    created_at TIMESTAMP DEFAULT NOW()
);