#!/bin/bash

BASE_URL="http://localhost:8081"
ENDPOINT="ping"

SUCCESS_COUNT=0
BLOCKED_COUNT=0

echo -e "\nMaking requests to $BASE_URL/$ENDPOINT...\n"

for i in {1..100}
do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/$ENDPOINT")

    if [ "$STATUS" -eq 200 ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT+1))
        # %3d forces the counter to always occupy 4 chars
        printf "[%3d] ✓ Status: 200 - Request Passed\n" "$i"
    elif [ "$STATUS" -eq 429 ]; then
        BLOCKED_COUNT=$((BLOCKED_COUNT+1))
        printf "[%3d] ✗ Status: 429 - Rate Limited\n" "$i"
    else
        printf "[%3d] ! Status: %s - Unexpected Response\n" "$i" "$STATUS"
    fi
done

echo -e "\n=============================================\n"
echo "Summary:"
echo "✓ Success (200): $SUCCESS_COUNT"
echo "✗ Blocked (429): $BLOCKED_COUNT"
echo -e "\n=============================================\n"
