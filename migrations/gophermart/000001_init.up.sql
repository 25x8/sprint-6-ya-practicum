-- Создаем таблицу для хранения пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    balance NUMERIC(10, 2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем таблицу для хранения информации о заказах
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    number VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL,
    accrual NUMERIC(10, 2) NOT NULL DEFAULT 0,
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем таблицу для хранения механик начисления баллов
CREATE TABLE IF NOT EXISTS mechanics (
    id SERIAL PRIMARY KEY,
    match TEXT UNIQUE NOT NULL,
    reward NUMERIC(10, 2) NOT NULL,
    reward_type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Создаем таблицу для хранения операций списания
CREATE TABLE IF NOT EXISTS withdrawals (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    order_number VARCHAR(255) NOT NULL,
    sum NUMERIC(10, 2) NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
); 