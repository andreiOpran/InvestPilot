package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// helper to setup security routes
func setupSecurityRouter(mockSvc *servicemocks.MockSecurityService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	handler := NewSecurityHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.GET("/2fa/setup", handler.Setup2FAHandler)
	r.POST("/2fa/enable", handler.Enable2FAHandler)

	return r
}

func TestSetup2FAHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockSecurityService)
	r := setupSecurityRouter(mockSvc)

	t.Run("Setup2FAHandler_alreadyEnabled_returns400", func(t *testing.T) {
		mockSvc.On("Setup2FA", uint(1)).Return("", "", "", services.Err2FAAlreadyEnabled).Once()

		req, _ := http.NewRequest(http.MethodGet, "/2fa/setup", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "already enabled")
	})

	t.Run("Setup2FAHandler_userNotFound_returns404", func(t *testing.T) {
		mockSvc.On("Setup2FA", uint(1)).Return("", "", "", services.ErrUserNotFound).Once()

		req, _ := http.NewRequest(http.MethodGet, "/2fa/setup", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Setup2FAHandler_internalError_returns500", func(t *testing.T) {
		mockSvc.On("Setup2FA", uint(1)).Return("", "", "", services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/2fa/setup", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Setup2FAHandler_success_returns200", func(t *testing.T) {
		mockSvc.On("Setup2FA", uint(1)).Return("secret", "otpauth://...", "base64qr", nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/2fa/setup", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "secret")
		assert.Contains(t, w.Body.String(), "data:image/png;base64,base64qr")
	})

	mockSvc.AssertExpectations(t)
}

func TestEnable2FAHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockSecurityService)
	r := setupSecurityRouter(mockSvc)

	t.Run("Enable2FAHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBufferString("bad-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Enable2FAHandler_userNotFound_returns404", func(t *testing.T) {
		payload := models.Enable2FARequest{Token: "111111"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Enable2FA", uint(1), "111111").Return(services.ErrUserNotFound).Once()

		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Enable2FAHandler_alreadyEnabled_returns400", func(t *testing.T) {
		payload := models.Enable2FARequest{Token: "222222"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Enable2FA", uint(1), "222222").Return(services.Err2FAAlreadyEnabled).Once()

		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "already enabled")
	})

	t.Run("Enable2FAHandler_internalError_returns500", func(t *testing.T) {
		payload := models.Enable2FARequest{Token: "333333"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Enable2FA", uint(1), "333333").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Enable2FAHandler_invalidToken_returns400", func(t *testing.T) {
		payload := models.Enable2FARequest{Token: "000000"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Enable2FA", uint(1), "000000").Return(services.ErrInvalid2FAToken).Once()

		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid code")
	})

	t.Run("Enable2FAHandler_success_returns200", func(t *testing.T) {
		payload := models.Enable2FARequest{Token: "123456"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Enable2FA", uint(1), "123456").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/2fa/enable", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "successfully enabled")
	})

	mockSvc.AssertExpectations(t)
}
