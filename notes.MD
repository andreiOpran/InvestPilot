
- could get info about account existing with a given email by spamming register.
  - we could change the message and always return 201 ({ "message": "If this email is not registered, an account has been created." })
  - we could implement confirmation email with SendGrid
  - limit /register to 5 attempt per Ip per hour

```
curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei@test.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

{
  "error": "Email already registered"
}
```

