
- could get info about account existing with a given email by spamming register.
  - we could change the message and always return 201 ({ "message": "If this email is not registered, an account has been created." })
  - we could implement confirmation email with SendGrid
  - limit /register to 5 attempt per IP per hour

```
curl -s -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d '{"email": "andrei@test.com", "password": "secure123", "risk_tolerance": 4, "investment_horizon": 10}' | jq .

{
  "error": "Email already registered"
}
```

- session timeout:
  - Redis variant
  - 24hr delay
  - user has an increment that constitutes a part of the token


- inspect db in bash:
```sql
docker exec -it licenta-db-1 psql -U admin -d robo_advisory

-- see all tables
\dt

-- see market data
SELECT * FROM historical_market_data LIMIT 20;

-- see table structure
\d historical_market_data

-- exit
\q
```

- adminer: http://localhost:8082/, server=db, username=admin, password=pass, database=robo_advisory
