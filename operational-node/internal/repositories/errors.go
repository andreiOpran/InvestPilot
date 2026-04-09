package repositories

import "errors"

var (
	ErrUserCashoutInsufficientFunds = errors.New("cashout value is greater than actual balance")
)
