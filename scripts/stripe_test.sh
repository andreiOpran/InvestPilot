# terminal 1:
# run command and then copy webhook signing secret (whsec_...) to STRIPE_WEBHOOK_KEY env var
stripe listen --forward-to localhost:8081/api/v1/webhook/stripe

# terminal 2:
# first get payment intend from /deposit/intent (pi_3TMvNaK6SZHKi0bb1YuhMoaN)
stripe payment_intents confirm pi_3TMvNaK6SZHKi0bb1YuhMoaN --payment-method pm_card_visa