
- could get info about account existing with a given email:

```
curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei@test.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

{
  "error": "Email already registered"
}
```

