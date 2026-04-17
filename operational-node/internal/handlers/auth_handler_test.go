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
	r.POST("/verify-2fa", handler.Verify2FAHandler)
	r.POST("/refresh-token", handler.RefreshTokenHandler)
	r.POST("/reset-password", handler.ResetPasswordHandler)
	r.POST("/logout", handler.LogoutHandler)

	return r
}

func TestRegisterHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("RegisterHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// TODO: adapt because we no longer have ErrEmailExists, we pretend it worked
	// t.Run("RegisterHandler_emailExists_returns409", func(t *testing.T) {
	// 	payload := models.RegisterRequest{Email: "test@test.com", Password: "password123"}
	// 	body, _ := json.Marshal(payload)

	// 	mockSvc.On("RegisterUser", payload).Return(services.ErrEmailExists).Once()

	// 	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	// 	req.Header.Set("Content-Type", "application/json")
	// 	w := httptest.NewRecorder()
	// 	r.ServeHTTP(w, req)

	// 	assert.Equal(t, http.StatusConflict, w.Code)
	// 	assert.Contains(t, w.Body.String(), "already registered")
	// })

	t.Run("RegisterHandler_internalError_returns500", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "fail@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RegisterUser", payload).Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	t.Run("LoginHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

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

	t.Run("LoginHandler_internalError_returns500", func(t *testing.T) {
		payload := models.LoginRequest{Email: "error@test.com", Password: "password123"}
		body, _ := json.Marshal(payload)
		mockSvc.On("AuthenticateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*services.LoginResult)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	t.Run("ForgotPasswordHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

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

	t.Run("Verify2FAHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Verify2FAHandler_invalidCredentials_returns401", func(t *testing.T) {
		// Adaugat Password si Token pentru a trece de validarea GIN
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrInvalidCredentials).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		localMock.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_notEnabled_returns400", func(t *testing.T) {
		// Adaugat Password si Token pentru a trece de validarea GIN
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.Err2FANotEnabled).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		localMock.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_invalidToken_returns401", func(t *testing.T) {
		// Adaugat Password si Token pentru a trece de validarea GIN
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrInvalid2FAToken).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		localMock.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_success_returns200", func(t *testing.T) {
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("Verify2FA", "test@test.com", "pass", "123456", mock.Anything, mock.Anything).Return("acc", "ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		localMock.AssertExpectations(t)
	})
}

func TestRefreshTokenHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("RefreshTokenHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RefreshTokenHandler_invalidToken_returns401", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "bad"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RefreshToken", mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrTokenInvalid).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RefreshTokenHandler_reuseDetected_returns401", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "reused"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RefreshToken", mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrTokenReuseDetected).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("RefreshTokenHandler_concurrentRequest_returns409", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "conc"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RefreshToken", mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrConcurrentRequest).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("RefreshTokenHandler_success_returns200", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "old-token"}
		body, _ := json.Marshal(payload)

		mockSvc.On("RefreshToken", "old-token", mock.Anything, mock.Anything).Return("new-acc", "new-ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockSvc.AssertExpectations(t)
}

func TestResetPasswordHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("ResetPasswordHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ResetPasswordHandler_invalidToken_returns400", func(t *testing.T) {
		// Adaugat NewPassword pentru a trece de validarea GIN
		payload := models.ResetPasswordRequest{Token: "bad", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("ResetPassword", mock.Anything, mock.Anything).Return(services.ErrTokenInvalid).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		localMock.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_expiredToken_returns400", func(t *testing.T) {
		// Adaugat NewPassword pentru a trece de validarea GIN
		payload := models.ResetPasswordRequest{Token: "exp", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("ResetPassword", mock.Anything, mock.Anything).Return(services.ErrTokenExpired).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		localMock.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_success_returns200", func(t *testing.T) {
		payload := models.ResetPasswordRequest{Token: "token123", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)

		localMock := new(servicemocks.MockAuthService)
		localRouter := setupAuthRouter(localMock)

		localMock.On("ResetPassword", "token123", "new_password").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		localRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		localMock.AssertExpectations(t)
	})
}
func TestLogoutHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockAuthService)
	r := setupAuthRouter(mockSvc)

	t.Run("LogoutHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("LogoutHandler_success_returns200", func(t *testing.T) {
		payload := models.RefreshRequest{RefreshToken: "token123"}
		body, _ := json.Marshal(payload)

		mockSvc.On("LogoutUser", "token123").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockSvc.AssertExpectations(t)
}
