package router

import (
	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/handlers"
	"github.com/andreiOpran/licenta/operational-node/internal/middleware"
)

// SetupRoutes registers all HTTP endpoints and maps them to handler functions
func SetupRoutes(r *gin.Engine) {
	r.GET("/ping", handlers.PingHandler)
	r.GET("/status", handlers.StatusHandler)
	r.GET("/test-email", handlers.TestEmailHandler)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/simulate-investment", handlers.SimulateInvestmentHandler)

		v1.POST("/register", handlers.RegisterHandler)
		v1.GET("/verify-email", handlers.VerifyEmailHandler)
		v1.POST("/login", handlers.LoginHandler)
		v1.POST("/logout", handlers.LogoutHandler)
		v1.POST("/verify-2fa", handlers.Verify2FAHandler)
		v1.POST("/refresh-token", handlers.RefreshTokenHandler)
		v1.POST("/forgot-password", handlers.ForgotPasswordHandler)
		v1.POST("/reset-password", handlers.ResetPasswordHandler)

		protected := v1.Group("/", middleware.AuthMiddleware())
		{
			protected.GET("/user", handlers.GetUserHandler)
			protected.GET("/2fa/setup", handlers.Setup2FAHandler)
			protected.POST("/2fa/enable", handlers.Enable2FAHandler)
			protected.POST("/deposit", handlers.DepositHandler)
		}
	}
}
