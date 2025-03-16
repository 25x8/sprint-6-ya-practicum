-- Создаем таблицу для хранения информации о заказах
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    number TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL,
    accrual NUMERIC(10, 2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Создаем таблицу для хранения механик начисления баллов
CREATE TABLE IF NOT EXISTS mechanics (
    id SERIAL PRIMARY KEY,
    match TEXT UNIQUE NOT NULL,
    reward NUMERIC(10, 2) NOT NULL,
    reward_type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
); 