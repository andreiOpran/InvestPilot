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

func setupUserRouter(mockService *servicemocks.MockUserService) (*gin.Engine, *UserHandler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewUserHandler(mockService)

	authGroup := r.Group("/")
	authGroup.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	authGroup.GET("/user", handler.GetUserHandler)
	authGroup.PUT("/user/profile", handler.UpdateProfileHandler)
	authGroup.POST("/deposit", handler.DepositHandler)
	authGroup.POST("/cashout", handler.CashoutHandler)

	return r, handler
}

func TestGetUserHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockUserService)
	r, _ := setupUserRouter(mockSvc)

	t.Run("GetUserHandler_userNotFound_returns404", func(t *testing.T) {
		mockSvc.On("GetUserProfile", uint(1)).Return((*models.User)(nil), services.ErrUserNotFound).Once()

		req, _ := http.NewRequest(http.MethodGet, "/user", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not found")
	})

	t.Run("GetUserHandler_internalError_returns500", func(t *testing.T) {
		mockSvc.On("GetUserProfile", uint(1)).Return((*models.User)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/user", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("GetUserHandler_success_returns200", func(t *testing.T) {
		expectedUser := &models.User{Email: "test@example.com", Wallet: models.Wallet{Balance: 150.5}}
		mockSvc.On("GetUserProfile", uint(1)).Return(expectedUser, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/user", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test@example.com")
		assert.Contains(t, w.Body.String(), "150.5")
	})

	mockSvc.AssertExpectations(t)
}

func TestUpdateProfileHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockUserService)
	r, _ := setupUserRouter(mockSvc)

	t.Run("UpdateProfileHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer([]byte("{bad json}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("UpdateProfileHandler_userNotFound_returns404", func(t *testing.T) {
		payload := models.UpdateProfileRequest{RiskTolerance: 4, InvestmentHorizon: 20}
		body, _ := json.Marshal(payload)
		mockSvc.On("UpdateUserProfile", uint(1), payload).Return(services.ErrUserNotFound).Once()

		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("UpdateProfileHandler_success_returns200", func(t *testing.T) {
		payload := models.UpdateProfileRequest{RiskTolerance: 4, InvestmentHorizon: 20}
		body, _ := json.Marshal(payload)
		mockSvc.On("UpdateUserProfile", uint(1), payload).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "updated successfully")
	})

	mockSvc.AssertExpectations(t)
}

func TestDepositHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockUserService)
	r, _ := setupUserRouter(mockSvc)

	t.Run("DepositHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/deposit", bytes.NewBuffer([]byte("{bad json}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("DepositHandler_userNotFound_returns404", func(t *testing.T) {
		payload := models.DepositRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("DepositFunds", uint(1), 100.0).Return(0.0, services.ErrUserNotFound).Once()

		req, _ := http.NewRequest(http.MethodPost, "/deposit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DepositHandler_success_returns200", func(t *testing.T) {
		payload := models.DepositRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("DepositFunds", uint(1), 100.0).Return(150.0, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/deposit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "150")
	})

	mockSvc.AssertExpectations(t)
}

func TestCashoutHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockUserService)
	r, _ := setupUserRouter(mockSvc)

	t.Run("CashoutHandler_badJSON_returns400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/cashout", bytes.NewBuffer([]byte("{bad json}")))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("CashoutHandler_insufficientBalance_returns400", func(t *testing.T) {
		payload := models.CashoutRequest{Amount: 9999.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Cashout", uint(1), 9999.0).Return(0.0, services.ErrInsufficientBalance).Once()

		req, _ := http.NewRequest(http.MethodPost, "/cashout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Insufficient")
	})

	t.Run("CashoutHandler_success_returns200", func(t *testing.T) {
		payload := models.CashoutRequest{Amount: 50.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Cashout", uint(1), 50.0).Return(50.0, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/cashout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "successfully")
	})

	mockSvc.AssertExpectations(t)
}
