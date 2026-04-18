package middleware

import "errors"

var (
	ErrTooManyRequests = errors.New("too many requests. please slow down and try again later")
)
