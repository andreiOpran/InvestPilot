#!/bin/bash


DB_CONTAINER="licenta-db-1"
DB_USER="admin"
DB_NAME="robo_advisory"


docker exec -i $DB_CONTAINER psql -U $DB_USER -d $DB_NAME <<EOF

-- 1. Create User
INSERT INTO users (id, email, password, is_email_verified, investment_horizon, risk_tolerance, created_at, updated_at) VALUES
(104, 'history_test@test.com', '$2a$14$lpRzO8DPHOqP5./N0mK.geXl.8fV/5p5rG7Xk7y6Yk6Z8uG7yD5q.', true, 10, 4, '2022-01-01', NOW());

-- 2. Create Wallet
INSERT INTO wallets (user_id, balance, created_at, updated_at) VALUES
(104, 0.00, '2022-01-01', NOW());

-- 3. Create historical deposits and withdrawals to affect "Net Contributions" dynamically over time
INSERT INTO fundings (user_id, type, amount, status, created_at, updated_at) VALUES
(104, 'DEPOSIT', 1000.00, 'COMPLETED', '2022-01-05', NOW()),
(104, 'DEPOSIT', 2000.00, 'COMPLETED', '2023-06-15', NOW()),
(104, 'WITHDRAWAL', 500.00, 'COMPLETED', '2024-02-20', NOW()),
(104, 'DEPOSIT', 5000.00, 'COMPLETED', '2025-11-10', NOW());

-- 4. Create sequential Investment Rounds matching the funding events
-- Only the last round is active
INSERT INTO investment_rounds (id, user_id, total_value, is_active, created_at) VALUES
(401, 104, 1000.00, false, '2022-01-06'),
(402, 104, 3000.00, false, '2023-06-16'),
(403, 104, 2500.00, false, '2024-02-21'),
(404, 104, 7500.00, true, '2025-11-11');

-- 5. Seed Holdings to see portfolio composition changes over time

-- 2022 ROUND (VTI + USD)
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(104, 401, 'VTI', 0.50, 2.50, 200.0, 500.00, '2022-01-06', NOW()),
(104, 401, 'USD', 0.50, 500.00, 1.0, 500.00, '2022-01-06', NOW());

-- 2023 ROUND (Added QQQ)
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(104, 402, 'VTI', 0.40, 6.00, 200.0, 1200.00, '2023-06-16', NOW()),
(104, 402, 'QQQ', 0.40, 4.00, 300.0, 1200.00, '2023-06-16', NOW()),
(104, 402, 'USD', 0.20, 600.00, 1.0, 600.00, '2023-06-16', NOW());

-- 2024 ROUND (Sold some after withdrawal)
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(104, 403, 'VTI', 0.50, 6.25, 200.0, 1250.00, '2024-02-21', NOW()),
(104, 403, 'QQQ', 0.30, 2.50, 300.0, 750.00, '2024-02-21', NOW()),
(104, 403, 'USD', 0.20, 500.00, 1.0, 500.00, '2024-02-21', NOW());

-- 2025 ROUND (Major deposit, added BND)
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(104, 404, 'VTI', 0.40, 15.00, 200.0, 3000.00, '2025-11-11', NOW()),
(104, 404, 'QQQ', 0.40, 10.00, 300.0, 3000.00, '2025-11-11', NOW()),
(104, 404, 'BND', 0.10, 10.00, 75.0, 750.00, '2025-11-11', NOW()),
(104, 404, 'USD', 0.10, 750.00, 1.0, 750.00, '2025-11-11', NOW());

EOF