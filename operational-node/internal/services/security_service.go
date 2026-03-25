package services

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/pquerna/otp/totp"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
)

type SecurityService interface {
	Setup2FA(userID uint) (string, string, string, error)
	Enable2FA(userID uint, token string) error
}

type securityService struct {
	userRepo repositories.UserRepository
}

func NewSecurityService(userRepo repositories.UserRepository) SecurityService {
	return &securityService{
		userRepo: userRepo,
	}
}

func (s *securityService) Setup2FA(userID uint) (string, string, string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", "", "", ErrUserNotFound
	}

	if user.IsTwoFactorEnable {
		return "", "", "", Err2FAAlreadyEnabled
	}

	// generate OTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Robo-Advisory",
		AccountName: user.Email,
	})
	if err != nil {
		return "", "", "", ErrInternal
	}

	encryptedSecret, err := crypto.EncryptAES(key.Secret(), []byte(config.Env.AESMasterKey))
	if err != nil {
		return "", "", "", ErrInternal
	}
	// temp save secret (user must confirm it to enable)
	user.TwoFactorSecret = encryptedSecret
	s.userRepo.Save(user)

	// generate QR code image
	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", "", ErrInternal
	}

	png.Encode(&buf, img)
	b64String := base64.StdEncoding.EncodeToString(buf.Bytes())

	return key.Secret(), key.URL(), b64String, nil
}

func (s *securityService) Enable2FA(userID uint, token string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.IsTwoFactorEnable {
		return Err2FAAlreadyEnabled
	}

	plainSecret, err := crypto.DecryptAES(user.TwoFactorSecret, []byte(config.Env.AESMasterKey))
	if err != nil {
		return ErrInternal
	}

	// validate the code agains the secret we saved during /setup
	valid := totp.Validate(token, plainSecret)
	if !valid {
		return ErrInvalid2FAToken
	}

	user.IsTwoFactorEnable = true
	s.userRepo.Save(user)

	return nil
}
