package token

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// helper for generating random string
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// helper for generating short-lived Access JWT and a long lived Refresh Token
// jwtSecret is provided by callers to avoid package-level globals
func GenerateTokensAndSession(c *gin.Context, userID uint, jwtSecret []byte) (string, string, error) {
	// generate short-lived JWT (10 minutes)
	claims := models.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.Env.AccessTokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := tokenObj.SignedString(jwtSecret)
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
		ExpiresAt:    time.Now().Add(config.Env.RefreshTokenLifetime),
	}

	if err := database.DB.Create(&session).Error; err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
