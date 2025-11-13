package routes

import (
	"KPI_System_Backend/controllers"
	"KPI_System_Backend/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Controllers
	authController := controllers.NewAuthController(db)
	pingController := controllers.NewPingController()
	
	// Public routes
	public := router.Group("/api")
	{
		public.POST("/auth/login", authController.Login)
		public.GET("/ping", pingController.Ping)
	}
	
	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/auth/profile", authController.GetProfile)
		
		// Add other protected routes here
	}
}