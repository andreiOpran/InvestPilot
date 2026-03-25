package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// helper to setup auth routes
func setupAuthRouter(mockSvc *servicemocks.MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	handler := NewAuthHandler(mockSvc)

	r.POST("/register", handler.RegisterHandler)
	r.POST("/login", handler.LoginHandler)
	r.GET("/verify-email", handler.VerifyEmailHandler)
	r.POST("/forgot-password", handler.ForgotPasswordHandler)

	return r
}

func TestRegisterHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("RegisterHandler_emailExists_returns409", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "test@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)

		mockSvc.On("RegisterUser", payload).Return(services.ErrEmailExists).Once()

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already registered")
	})

	t.Run("RegisterHandler_success_returns200", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "new@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)

		mockSvc.On("RegisterUser", payload).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "verification link has been sent")
	})

	mockSvc.AssertExpectations(t)
}

func TestLoginHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("LoginHandler_invalidCredentials_returns401", func(t *testing.T) {
		payload := models.LoginRequest{Email: "test@test.com", Password: "wrong"}
		body, _ := json.Marshal(payload)

		mockSvc.On("AuthenticateUser", "test@test.com", "wrong", mock.Anything, mock.Anything).Return((*services.LoginResult)(nil), services.ErrInvalidCredentials).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid email or password")
	})

	t.Run("LoginHandler_success_returnsTokens", func(t *testing.T) {
		payload := models.LoginRequest{Email: "test@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)

		res := &services.LoginResult{Requires2FA: false, AccessToken: "acc_token", RefreshToken: "ref_token"}
		mockSvc.On("AuthenticateUser", "test@test.com", "password123", mock.Anything, mock.Anything).Return(res, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "acc_token")
		assert.Contains(t, w.Body.String(), "ref_token")
	})

	t.Run("LoginHandler_2FARequired_returnsStatus2FA", func(t *testing.T) {
		payload := models.LoginRequest{Email: "2fa@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)

		res := &services.LoginResult{Requires2FA: true, Email: "2fa@test.com"}
		mockSvc.On("AuthenticateUser", "2fa@test.com", "password123", mock.Anything, mock.Anything).Return(res, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "2fa_required")
	})

	mockSvc.AssertExpectations(t)
}

func TestVerifyEmailHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("VerifyEmailHandler_missingToken_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/verify-email", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("VerifyEmailHandler_success_returns200", func(t *testing.T) {
		mockSvc.On("VerifyEmail", "valid-token").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/verify-email?token=valid-token", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockSvc.AssertExpectations(t)
}

func TestForgotPasswordHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("ForgotPasswordHandler_success_returns200", func(t *testing.T) {
		payload := models.ForgotPasswordRequest{Email: "test@test.com"}
		body, _ := json.Marshal(payload)

		mockSvc.On("ForgotPassword", "test@test.com").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockSvc.AssertExpectations(t)
}

func TestVerify2FAHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)
	// need to register route manually for this test since it's not in setup helper
	r.POST("/verify-2fa", NewAuthHandler(mockSvc).Verify2FAHandler)

	t.Run("Verify2FAHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer([]byte("{bad}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Verify2FAHandler_success_returns200", func(t *testing.T) {
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)

		mockSvc.On("Verify2FA", "test@test.com", "pass", "123456", mock.Anything, mock.Anything).Return("acc", "ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRefreshTokenHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)
	r.POST("/refresh-token", NewAuthHandler(mockSvc).RefreshTokenHandler)

	t.Run("RefreshTokenHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer([]byte("{bad}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RefreshTokenHandler_success_returns200", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "old-token"}
		body, _ := json.Marshal(payload)

		mockSvc.On("RefreshToken", "old-token", mock.Anything, mock.Anything).Return("new-acc", "new-ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestResetPasswordHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)
	r.POST("/reset-password", NewAuthHandler(mockSvc).ResetPasswordHandler)

	t.Run("ResetPasswordHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer([]byte("{bad}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ResetPasswordHandler_success_returns200", func(t *testing.T) {
		payload := models.ResetPasswordRequest{Token: "token123", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)

		mockSvc.On("ResetPassword", "token123", "new_password").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestLogoutHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)
	r.POST("/logout", NewAuthHandler(mockSvc).LogoutHandler)

	t.Run("LogoutHandler_success_returns200", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "token123"}
		body, _ := json.Marshal(payload)

		mockSvc.On("LogoutUser", "token123").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
