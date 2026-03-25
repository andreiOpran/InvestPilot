package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/handlers"
	"github.com/andreiOpran/licenta/operational-node/internal/middleware"
)

// SetupRoutes registers all HTTP endpoints and maps them to handler functions
func SetupRoutes(r *gin.Engine) {
	// setup cors middleware for frontend communication
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.Env.FrontendBaseURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
