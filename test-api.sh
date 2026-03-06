#!/bin/bash

BASE_URL="http://localhost:8081/api/v1"

echo "--- 1. Register ---"
curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei2@test.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

echo ""
echo "--- 2. Login ---"
TOKEN=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei2@test.com", "password": "secure123"}' | jq -r .token)

echo "Token: $TOKEN"

echo ""
echo "--- 3. Get user (with token) ---"
curl -s -X GET "$BASE_URL/user" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "--- 4. Deposit 1000 ---"
curl -s -X POST "$BASE_URL/deposit" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 1000}' | jq .

echo ""
echo "--- 5. Get user again (balance should be updated) ---"
curl -s -X GET "$BASE_URL/user" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "--- 6. Get user WITHOUT token (should get 401) ---"
curl -s -X GET "$BASE_URL/user" | jq .
