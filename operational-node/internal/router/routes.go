package router

import (
	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/handlers"
	"github.com/andreiOpran/licenta/operational-node/internal/middleware"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// SetupRoutes registers all HTTP endpoints and maps them to handler functions
func SetupRoutes(r *gin.Engine) {
	// apply global middlewares
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	// init repositories
	authRepo := repositories.NewAuthRepository(database.DB)
	userRepo := repositories.NewUserRepository(database.DB)

	// init services with repository deps
	authService := services.NewAuthService(authRepo)
	userService := services.NewUserService(userRepo)
	securityService := services.NewSecurityService(userRepo)

	// init handlers with service deps
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	securityHandler := handlers.NewSecurityHandler(securityService)

	// standalone routes (no db required)
	r.GET("/ping", handlers.PingHandler)
	r.GET("/status", handlers.StatusHandler)
	r.GET("/test-email", handlers.TestEmailHandler)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/simulate-investment", handlers.SimulateInvestmentHandler)

		v1.POST("/register", authHandler.RegisterHandler)
		v1.GET("/verify-email", authHandler.VerifyEmailHandler)
		v1.POST("/login", authHandler.LoginHandler)
		v1.POST("/logout", authHandler.LogoutHandler)
		v1.POST("/verify-2fa", authHandler.Verify2FAHandler)
		v1.POST("/refresh-token", authHandler.RefreshTokenHandler)
		v1.POST("/forgot-password", authHandler.ForgotPasswordHandler)
		v1.POST("/reset-password", authHandler.ResetPasswordHandler)

		protected := v1.Group("/", middleware.AuthMiddleware())
		{
			protected.GET("/user", userHandler.GetUserHandler)
			protected.PUT("/user/profile", userHandler.UpdateProfileHandler)
			protected.GET("/2fa/setup", securityHandler.Setup2FAHandler)
			protected.POST("/2fa/enable", securityHandler.Enable2FAHandler)
			protected.POST("/deposit", userHandler.DepositHandler)
		}
	}
}
