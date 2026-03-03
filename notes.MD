
- could get info about account existing with a given email:

```curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei@test.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

{
  "error": "Email already registered"
}
```

- problem:
```
--- 6. Get user WITHOUT token (should get 401) ---
curl -X GET http://localhost:8081/api/v1/user

{
  "email": "test@roboadvisor.com",
  "investment_horizon": 10,
  "risk_tolerance": 4,
  "user_id": 1,
  "wallet_balance": 12001
}
```
