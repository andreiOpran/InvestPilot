#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# CONFIGURATION
# ==========================================
BASE_URL="http://localhost:8081/api/v1" # Adăugat prefixul corect
EMAIL="forecast_test_$RANDOM@test.com"
PASSWORD="password123"

INITIAL_INVESTMENT=10000
MONTHLY_CONTRIBUTION=500
YEARS=20

TIMEOUT=120
INTERVAL=2

DB_CONTAINER="licenta-db-1"
DB_USER="admin"
DB_NAME="robo_advisory"

echo "=========================================="
echo "1. Registering User via Go API"
echo "=========================================="

REGISTER_PAYLOAD=$(jq -n --arg email "$EMAIL" --arg password "$PASSWORD" '{email:$email, password:$password}')

# Prindem atât output-ul, cât și codul HTTP pentru siguranță
HTTP_STATUS=$(curl -s -o /tmp/reg_out.txt -w "%{http_code}" -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "$REGISTER_PAYLOAD")

if [ "$HTTP_STATUS" -ne 200 ] && [ "$HTTP_STATUS" -ne 201 ]; then
  echo "X Error: Registration failed with HTTP $HTTP_STATUS"
  cat /tmp/reg_out.txt
  echo
  exit 1
fi

echo "! User registered successfully with email: $EMAIL"
echo

echo "=========================================="
echo "2. SQL Injection: Activating Email & Seeding Portfolio"
echo "=========================================="

# Modificat tabelele conform GORM (holdings și investment_round_id)
docker exec -i $DB_CONTAINER psql -U $DB_USER -d $DB_NAME <<EOF
-- 1. Activate User and set Risk Profile
UPDATE users SET is_email_verified = true, investment_horizon = 20, risk_tolerance = 5 WHERE email = '${EMAIL}';

-- 2. Create Wallet
INSERT INTO wallets (user_id, balance, created_at, updated_at)
VALUES ((SELECT id FROM users WHERE email = '${EMAIL}'), 0.00, NOW(), NOW());

-- 3. Create Active Investment Round
INSERT INTO investment_rounds (user_id, total_value, is_active, created_at)
VALUES ((SELECT id FROM users WHERE email = '${EMAIL}'), 10000.00, true, NOW());

-- 4. Insert Holdings (So Monte Carlo has tickers to simulate)
INSERT INTO holdings (user_id, investment_round_id, ticker, weight, shares, purchase_price, allocated_amount, created_at, updated_at) VALUES
((SELECT id FROM users WHERE email = '${EMAIL}'), (SELECT id FROM investment_rounds WHERE user_id = (SELECT id FROM users WHERE email = '${EMAIL}') AND is_active = true), 'QQQ', 0.60, 10.0, 300.0, 6000.00, NOW(), NOW()),
((SELECT id FROM users WHERE email = '${EMAIL}'), (SELECT id FROM investment_rounds WHERE user_id = (SELECT id FROM users WHERE email = '${EMAIL}') AND is_active = true), 'VTI', 0.40, 20.0, 200.0, 4000.00, NOW(), NOW());
EOF

echo "! Email verified and 10,000 USD portfolio seeded!"
echo

echo "=========================================="
echo "3. Authenticating User"
echo "=========================================="

LOGIN_PAYLOAD=$(jq -n --arg email "$EMAIL" --arg password "$PASSWORD" '{email:$email, password:$password}')
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "$LOGIN_PAYLOAD")

TOKEN=$(echo "$LOGIN_RESP" | jq -r '.access_token // empty')

if [ -z "$TOKEN" ]; then
  echo "X Login failed. API Response:"
  echo "$LOGIN_RESP" | jq .
  exit 1
fi

echo "! Authentication successful! Token obtained."
echo

echo "=========================================="
echo "4. Requesting Monte Carlo Forecast"
echo "=========================================="

FORECAST_PAYLOAD=$(jq -n \
  --argjson initial "$INITIAL_INVESTMENT" \
  --argjson monthly "$MONTHLY_CONTRIBUTION" \
  --argjson years "$YEARS" \
  '{initial_investment: $initial, monthly_contribution: $monthly, years: $years}')

FORECAST_RESP=$(curl -s -X POST "$BASE_URL/forecast" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$FORECAST_PAYLOAD")

TASK_ID=$(echo "$FORECAST_RESP" | jq -r '.task_id // empty')

if [ -z "$TASK_ID" ]; then
  echo "X Failed to start forecast task. API Response:"
  echo "$FORECAST_RESP" | jq .
  exit 1
fi

echo "! Forecast task initiated! Task ID: $TASK_ID"
echo

echo "=========================================="
echo "5. Polling for Completion"
echo "=========================================="

START_TS=$(date +%s)
STATUS_URL="$BASE_URL/forecast/status/$TASK_ID"

while true; do
  STATUS_RESP=$(curl -s -H "Authorization: Bearer $TOKEN" "$STATUS_URL")
  STATUS=$(echo "$STATUS_RESP" | jq -r '.status // empty')

  echo "[$(date +%T)] Status: $STATUS"

  # Exit loop when Python engine finishes calculating
  if [ "$STATUS" != "pending" ]; then
    echo
    echo "=========================================="
    echo "6. Final Result Received"
    echo "=========================================="
    echo "$STATUS_RESP" | jq .
    exit 0
  fi

  # Handle Timeout
  NOW_TS=$(date +%s)
  ELAPSED=$((NOW_TS - START_TS))
  
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo "X Error: Polling timed out after ${TIMEOUT} seconds."
    exit 2
  fi

  sleep "$INTERVAL"
done