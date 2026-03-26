#!/bin/bash

BASE_URL="http://localhost:8081/api/v1"

curl -s -X POST "$BASE_URL/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei.opran@icloud.com"}' | jq .

curl -s -X POST "$BASE_URL/reset-password" \
-H "Content-Type: application/json" \
-d '{
"token": "2e1e5a1e4b8394fd72e5a07c31f2466dab8bffbf77b317ca8471f49abb2a8f6f",
"new_password": "newsecure456"
}' | jq .

curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "andrei.opran@icloud.com",
    "password": "newsecure456"
  }' | jq .

curl -s -X POST "$BASE_URL/verify-2fa" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "andrei.opran@icloud.com",
    "password": "newsecure456",
    "token": "123456"
  }' | jq .