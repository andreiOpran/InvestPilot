package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/andreiOpran/licenta/operational-node/utils/validator"
)

func init() {
	config.Env = config.AppSettings{
		TurnstileSecretKey:              "",
		RefreshTokenLifetimeSecondsInt:  604800,
		CookieDomain:                    "localhost",
		CookieSecure:                    false,
	}
}

func setupAuthRouter(mockSvc *servicemocks.MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
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

	t.Run("RegisterHandler_missingTurnstileToken_returns400", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "new@test.com", Password: "ValidPass1!"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RegisterHandler_weakPassword_returns400", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "wp@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RegisterUser", payload).Return(validator.ErrPasswordTooShort).Once()

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RegisterHandler_internalError_returns500", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "fail@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		mockSvc.On("RegisterUser", payload).Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("RegisterHandler_success_returns200", func(t *testing.T) {
		payload := models.RegisterRequest{Email: "new@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
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

	t.Run("LoginHandler_missingTurnstileToken_returns400", func(t *testing.T) {
		payload := models.LoginRequest{Email: "test@test.com", Password: "ValidPass1!"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("LoginHandler_invalidCredentials_returns401", func(t *testing.T) {
		payload := models.LoginRequest{Email: "test@test.com", Password: "wrong", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		mockSvc.On("AuthenticateUser", "test@test.com", "wrong", mock.Anything, mock.Anything).Return((*services.LoginResult)(nil), services.ErrInvalidCredentials).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("LoginHandler_accountLocked_returns429", func(t *testing.T) {
		payload := models.LoginRequest{Email: "locked@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		mockSvc.On("AuthenticateUser", "locked@test.com", "ValidPass1!", mock.Anything, mock.Anything).Return((*services.LoginResult)(nil), services.ErrAccountLocked).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("LoginHandler_internalError_returns500", func(t *testing.T) {
		payload := models.LoginRequest{Email: "error@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		mockSvc.On("AuthenticateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*services.LoginResult)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("LoginHandler_success_returnsAccessToken", func(t *testing.T) {
		payload := models.LoginRequest{Email: "ok@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		res := &services.LoginResult{Requires2FA: false, AccessToken: "acc_token", RefreshToken: "ref_token"}
		mockSvc.On("AuthenticateUser", "ok@test.com", "ValidPass1!", mock.Anything, mock.Anything).Return(res, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "acc_token")
		// refresh token goes in Set-Cookie header, not body
		found := false
		for _, c := range w.Result().Cookies() {
			if c.Name == "refresh_token" && c.Value == "ref_token" {
				found = true
			}
		}
		assert.True(t, found, "refresh_token cookie should be set")
	})

	t.Run("LoginHandler_2FARequired_returnsStatus2FA", func(t *testing.T) {
		payload := models.LoginRequest{Email: "2fa@test.com", Password: "ValidPass1!", TurnstileToken: "test-token"}
		body, _ := json.Marshal(payload)
		res := &services.LoginResult{Requires2FA: true, Email: "2fa@test.com"}
		mockSvc.On("AuthenticateUser", "2fa@test.com", "ValidPass1!", mock.Anything, mock.Anything).Return(res, nil).Once()

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

	t.Run("VerifyEmailHandler_invalidToken_returns400", func(t *testing.T) {
		mockSvc.On("VerifyEmail", "bad-token").Return(services.ErrTokenInvalid).Once()
		req, _ := http.NewRequest(http.MethodGet, "/verify-email?token=bad-token", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("VerifyEmailHandler_internalError_returns500", func(t *testing.T) {
		mockSvc.On("VerifyEmail", "err-token").Return(services.ErrInternal).Once()
		req, _ := http.NewRequest(http.MethodGet, "/verify-email?token=err-token", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	t.Run("ForgotPasswordHandler_missingTurnstileToken_returns400", func(t *testing.T) {
		payload := models.ForgotPasswordRequest{Email: "test@test.com"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ForgotPasswordHandler_success_returns200", func(t *testing.T) {
		payload := models.ForgotPasswordRequest{Email: "test@test.com", TurnstileToken: "test-token"}
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
	t.Run("Verify2FAHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Verify2FAHandler_invalidCredentials_returns401", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)
		mockSvc.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrInvalidCredentials).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_notEnabled_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)
		mockSvc.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.Err2FANotEnabled).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_invalidToken_returns401", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)
		mockSvc.On("Verify2FA", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", services.ErrInvalid2FAToken).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Verify2FAHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.Verify2FARequest{Email: "test@test.com", Password: "pass", Token: "123456"}
		body, _ := json.Marshal(payload)
		mockSvc.On("Verify2FA", "test@test.com", "pass", "123456", mock.Anything, mock.Anything).Return("acc", "ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/verify-2fa", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "acc")
		mockSvc.AssertExpectations(t)
	})
}

func TestRefreshTokenHandler(t *testing.T) {
	t.Run("RefreshTokenHandler_missingCookie_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RefreshTokenHandler_invalidToken_returns401", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		mockSvc.On("RefreshToken", "bad-token", mock.Anything, mock.Anything).Return("", "", services.ErrTokenInvalid).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad-token"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RefreshTokenHandler_reuseDetected_returns401", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		mockSvc.On("RefreshToken", "reused-token", mock.Anything, mock.Anything).Return("", "", services.ErrTokenReuseDetected).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "reused-token"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RefreshTokenHandler_concurrentRequest_returns409", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		mockSvc.On("RefreshToken", "conc-token", mock.Anything, mock.Anything).Return("", "", services.ErrConcurrentRequest).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "conc-token"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RefreshTokenHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		mockSvc.On("RefreshToken", "old-token", mock.Anything, mock.Anything).Return("new-acc", "new-ref", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/refresh-token", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-token"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new-acc")
		mockSvc.AssertExpectations(t)
	})
}

func TestResetPasswordHandler(t *testing.T) {
	t.Run("ResetPasswordHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ResetPasswordHandler_invalidToken_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.ResetPasswordRequest{Token: "bad", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)
		mockSvc.On("ResetPassword", mock.Anything, mock.Anything).Return(services.ErrTokenInvalid).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_expiredToken_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.ResetPasswordRequest{Token: "exp", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)
		mockSvc.On("ResetPassword", mock.Anything, mock.Anything).Return(services.ErrTokenExpired).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_weakPassword_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.ResetPasswordRequest{Token: "tok", NewPassword: "weakpa"}
		body, _ := json.Marshal(payload)
		mockSvc.On("ResetPassword", "tok", "weakpa").Return(validator.ErrPasswordTooShort).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.ResetPasswordRequest{Token: "tok2", NewPassword: "pass22"}
		body, _ := json.Marshal(payload)
		mockSvc.On("ResetPassword", "tok2", "pass22").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("ResetPasswordHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		payload := models.ResetPasswordRequest{Token: "token123", NewPassword: "new_password"}
		body, _ := json.Marshal(payload)
		mockSvc.On("ResetPassword", "token123", "new_password").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestLogoutHandler(t *testing.T) {
	t.Run("LogoutHandler_noCookie_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Logged out")
	})

	t.Run("LogoutHandler_withCookie_callsLogoutAndReturns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockAuthService)
		r := setupAuthRouter(mockSvc)
		mockSvc.On("LogoutUser", "token123").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/logout", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token123"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

// unused time import guard
var _ = time.Now
