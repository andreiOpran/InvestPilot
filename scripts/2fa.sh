#!/bin/bash

BASE_URL="http://localhost:8081/api/v1"

curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei.opran@icloud.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

curl -X GET "$BASE_URL/verify-email?token=778224f663322bccd7a0bc6274bb32a521e322236c7aad3c928c1033db2fefc5" | jq .

curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei.opran@icloud.com", "password": "secure123"}' | jq .

curl -X GET "$BASE_URL/2fa/setup" \
     -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo5MSwiZXhwIjoxNzc0MzY4NDQ1LCJpYXQiOjE3NzQyODIwNDV9.Fpate0GAd6awdCUkhDNWRaQxAVd7cmU6lakOXANNodE" | jq .

curl -X POST "$BASE_URL/2fa/enable" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo5MSwiZXhwIjoxNzc0MzY4NDQ1LCJpYXQiOjE3NzQyODIwNDV9.Fpate0GAd6awdCUkhDNWRaQxAVd7cmU6lakOXANNodE" \
  -H "Content-Type: application/json" \
  -d '{"token": "123456"}' | jq .
