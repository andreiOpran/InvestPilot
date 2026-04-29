package turnstile

import "errors"

var (
	ErrInvalidCaptcha = errors.New("invalid captcha token")
)
