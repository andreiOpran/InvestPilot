#!/bin/bash


DB_CONTAINER="licenta-db-1"
DB_USER="admin"
DB_NAME="robo_advisory"


docker exec -i $DB_CONTAINER psql -U $DB_USER -d $DB_NAME <<EOF

-- 1. Create 3 Test Users
-- Passwords are 'password123' (bcrypt hashed)
INSERT INTO users (id, email, password, is_email_verified, investment_horizon, risk_tolerance, created_at, updated_at) VALUES
(101, 'user_low_risk@test.com', '$2a$14$lpRzO8DPHOqP5./N0mK.geXl.8fV/5p5rG7Xk7y6Yk6Z8uG7yD5q.', true, 1, 1, NOW(), NOW()),
(102, 'user_med_risk@test.com', '$2a$14$lpRzO8DPHOqP5./N0mK.geXl.8fV/5p5rG7Xk7y6Yk6Z8uG7yD5q.', true, 5, 3, NOW(), NOW()),
(103, 'user_high_risk@test.com', '$2a$14$lpRzO8DPHOqP5./N0mK.geXl.8fV/5p5rG7Xk7y6Yk6Z8uG7yD5q.', true, 15, 5, NOW(), NOW());

-- 2. Create Wallets
INSERT INTO wallets (user_id, balance, created_at, updated_at) VALUES
(101, 50.00, NOW(), NOW()),
(102, 500.00, NOW(), NOW()),
(103, 0.00, NOW(), NOW());

-- 3. Create active Investment Rounds
INSERT INTO investment_rounds (id, user_id, total_value, is_active, created_at) VALUES
(201, 101, 1000.00, true, NOW()),
(202, 102, 5000.00, true, NOW()),
(203, 103, 10000.00, true, NOW());

-- 4. Seed Holdings with "Drift"
-- Tickers match your config: VTI, VOO, QQQ, BND, TLT

-- USER 101: Low Risk (Mostly Bonds). Weighted heavily in USD (50%) to test "Cash-First" rule.
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(101, 201, 'USD', 0.50, 500.00, 1.0, 500.00, NOW(), NOW()),
(101, 201, 'BND', 0.40, 5.71, 70.0, 400.00, NOW(), NOW()),
(101, 201, 'VTI', 0.10, 0.45, 220.0, 100.00, NOW(), NOW());

-- USER 102: Med Risk. Balanced but drifted. VOO is performing too well (high weight), BND is underweight.
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(102, 202, 'VOO', 0.60, 7.50, 400.0, 3000.00, NOW(), NOW()),
(102, 202, 'BND', 0.20, 14.28, 70.0, 1000.00, NOW(), NOW()),
(102, 202, 'TLT', 0.15, 7.50, 100.0, 750.00, NOW(), NOW()),
(102, 202, 'USD', 0.05, 250.00, 1.0, 250.00, NOW(), NOW());

-- USER 103: High Risk (Aggressive Equities). Perfectly balanced to test "Threshold Skip".
-- If drift is < 0.02 (your config), Python should return empty adjusted_targets for this user.
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
(103, 203, 'QQQ', 0.50, 15.15, 330.0, 5000.00, NOW(), NOW()),
(103, 203, 'VTI', 0.48, 21.81, 220.0, 4800.00, NOW(), NOW()),
(103, 203, 'USD', 0.02, 200.00, 1.0, 200.00, NOW(), NOW());

EOF

