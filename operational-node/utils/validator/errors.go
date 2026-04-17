package validator

import "errors"

var (
	ErrPasswordTooShort  = errors.New("password must be at least 10 characters long")
	ErrPasswordTooLong   = errors.New("password must be at most 128 characters long")
	ErrPasswordNoUpper   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber  = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")
	ErrPasswordTooCommon = errors.New("password is too weak, easily guessable, or commonly used")
)
