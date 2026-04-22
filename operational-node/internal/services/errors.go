package services

import "errors"

var (
	ErrUserNotFound                   = errors.New("user not found")
	ErrInvalidCredentials             = errors.New("invalid email or password")
	ErrAccountLocked                  = errors.New("account is temporarily locked due to multiple failed login attempts")
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
	ErrAmountNegative                 = errors.New("amount must be positive")
	ErrRebalancePausedStaleMarketData = errors.New("rebalance paused because market data is stale")
	ErrForecastUserNoActivePortfolio  = errors.New("user has no active portfolio to forecast")
	ErrForecastNoAssetsOnlyCash       = errors.New("portfolio consists entirely of uninvested cash; cannot forecast")
	ErrMissingAnswer                  = errors.New("missing answer for a required question")
	ErrInvalidOption                  = errors.New("invalid option selected for a question")
)
