package services

import "errors"

var (
	ErrUserNotFound                   = errors.New("user not found")
	ErrEmailExists                    = errors.New("email already registered")
	ErrInvalidCredentials             = errors.New("invalid email or password")
	ErrTokenExpired                   = errors.New("token has expired")
	ErrTokenInvalid                   = errors.New("invalid or expired token")
	Err2FAAlreadyEnabled              = errors.New("2FA is already enabled")
	Err2FANotEnabled                  = errors.New("2FA is not enabled on this account")
	ErrInvalid2FAToken                = errors.New("invalid 2FA token")
	ErrConcurrentRequest              = errors.New("concurrent request detected")
	ErrTokenReuseDetected             = errors.New("token reuse detected")
	ErrInternal                       = errors.New("internal server error")
	ErrBadRequest                     = errors.New("bad request parameters")
	ErrInsufficientBalance            = errors.New("insufficient wallet balance")
	ErrRebalancePausedStaleMarketData = errors.New("rebalance paused because market data is stale")
)
