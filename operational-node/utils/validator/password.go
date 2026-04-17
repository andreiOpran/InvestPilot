package validator

import (
	"errors"
	"unicode"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/nbutton23/zxcvbn-go"
)

// IsPasswordValidationError checks if a given
// error falls under the password policy umbrella
func IsPasswordValidationError(err error) bool {
	return errors.Is(err, ErrPasswordTooShort) ||
		errors.Is(err, ErrPasswordTooLong) ||
		errors.Is(err, ErrPasswordNoUpper) ||
		errors.Is(err, ErrPasswordNoLower) ||
		errors.Is(err, ErrPasswordNoNumber) ||
		errors.Is(err, ErrPasswordNoSpecial) ||
		errors.Is(err, ErrPasswordTooCommon)
}

// ValidatePassword checks the password policy using
// configured values together with a zxcvbn entropy check
func ValidatePassword(p string, userInputs []string) error {
	// min length constraint
	if len(p) < config.Env.PasswordMinLength {
		return ErrPasswordTooShort
	}
	// max length constraint
	if len(p) > config.Env.PasswordMaxLength {
		return ErrPasswordTooLong
	}

	// character class constraints
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range p {
		switch {
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	// return specific errors
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	// zxcvbn entropy check, passing also userInputs to avoid email/name as password
	// strength score from 0 to 4, we use configured config.Env.PasswordMinZxcvbnStrength
	strength := zxcvbn.PasswordStrength(p, userInputs)
	if strength.Score < config.Env.PasswordMinZxcvbnStrength {
		return ErrPasswordTooCommon
	}

	return nil
}
