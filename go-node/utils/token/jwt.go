package token

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"licenta/go-node/internal/database"
	"licenta/go-node/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TODO: in production should be retrieved from env var
var JwtSecret = []byte("secret-key")

// helper for generating random string
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// helper for generating short-lived Access JWT and a long lived Refresh Token
func GenerateTokensAndSession(c *gin.Context, userID uint) (string, string, error) {
	// generate short-lived JWT (10 minutes)
	claims := models.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := tokenObj.SignedString(JwtSecret)
	if err != nil {
		return "", "", err
	}

	// generate long-lived refresh token (consists of random hex available for 7 days)
	refreshToken, err := GenerateSecureToken(32)
	if err != nil {
		return "", "", err
	}

	// generate familyid for the token
	familyID, err := GenerateSecureToken(16)
	if err != nil {
		return "", "", err
	}

	// save session to database
	session := models.Session{
		UserID:       userID,
		FamilyID:     familyID,
		RefreshToken: refreshToken,
		IsUsed:       false,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	if err := database.DB.Create(&session).Error; err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
