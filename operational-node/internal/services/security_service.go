package services

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/pquerna/otp/totp"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
)

func Setup2FA(userID uint) (string, string, string, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
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
	database.DB.Save(&user)

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

func Enable2FA(userID uint, token string) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
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
	database.DB.Save(&user)

	return nil
}
