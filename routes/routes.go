package routes

import (
	"KPI_System_Backend/controllers"
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	authController := controllers.NewAuthController(db)
	pingController := controllers.NewPingController()
	divisionController := controllers.NewDivisionController(db)
	employeeController := controllers.NewEmployeeController(db)
	indicatorController := controllers.NewIndicatorController(db)
	periodController := controllers.NewEvaluationPeriodController(db)
	managerController := controllers.NewManagerController(db)
	positionController := controllers.NewPositionController(db)
	reportController := controllers.NewReportController(db)
	myPerformanceController := controllers.NewMyPerformanceController(db)

	public := router.Group("/api")
	{
		public.POST("/auth/login", authController.Login)
		public.GET("/ping", pingController.Ping)
	}

	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/auth/profile", authController.GetProfile)
		protected.PUT("/auth/change-password", authController.ChangePassword)

		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleAdmin))
		{
			adminRoutes.POST("/divisions", divisionController.CreateDivision)
			adminRoutes.GET("/divisions", divisionController.GetAllDivisions)
			adminRoutes.GET("/divisions/:id", divisionController.GetDivision)
			adminRoutes.PUT("/divisions/:id", divisionController.UpdateDivision)
			adminRoutes.DELETE("/divisions/:id", divisionController.DeleteDivision)

			adminRoutes.POST("/positions", positionController.CreatePosition)
			adminRoutes.GET("/positions", positionController.GetAllPositions)
			adminRoutes.PUT("/positions/:id", positionController.UpdatePosition)
			adminRoutes.DELETE("/positions/:id", positionController.DeletePosition)

			adminRoutes.POST("/employees", employeeController.CreateEmployee)
			adminRoutes.GET("/employees", employeeController.GetAllEmployees)
			adminRoutes.GET("/employees/:id", employeeController.GetEmployee)
			adminRoutes.PUT("/employees/:id", employeeController.UpdateEmployee)
			adminRoutes.DELETE("/employees/:id", employeeController.DeleteEmployee)

			adminRoutes.POST("/indicators", indicatorController.CreateIndicator)
			adminRoutes.GET("/indicators", indicatorController.GetAllIndicators)
			adminRoutes.GET("/indicators/:id", indicatorController.GetIndicator)
			adminRoutes.PUT("/indicators/:id", indicatorController.UpdateIndicator)
			adminRoutes.DELETE("/indicators/:id", indicatorController.DeleteIndicator)

			adminRoutes.POST("/periods", periodController.CreatePeriod)
			adminRoutes.GET("/periods", periodController.GetAllPeriods)
			adminRoutes.GET("/periods/:id", periodController.GetPeriod)
			adminRoutes.PUT("/periods/:id", periodController.UpdatePeriod)
			adminRoutes.DELETE("/periods/:id", periodController.DeletePeriod)
			adminRoutes.PATCH("/periods/:id/activate", periodController.SetActivePeriod)

			adminRoutes.GET("/reports/evaluations", reportController.GetEvaluationReport)
		}

		managerRoutes := protected.Group("/manager")
		managerRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleManager))
		{
			managerRoutes.GET("/my-team", managerController.GetMyTeam)
			managerRoutes.GET("/team-status", managerController.GetTeamEvaluationStatus)
			managerRoutes.POST("/evaluations/start", managerController.StartEvaluation)
			managerRoutes.GET("/evaluations/:id", managerController.GetEvaluationDetail)
			managerRoutes.PUT("/evaluations/:id/submit", managerController.SubmitEvaluation)
		}

		employeeRoutes := protected.Group("/employee")
		employeeRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleEmployee))
		{
			employeeRoutes.GET("/history", myPerformanceController.GetMyPerformanceHistory)
			employeeRoutes.GET("/latest", myPerformanceController.GetMyLatestPerformance)
			employeeRoutes.GET("/evaluations/:id", myPerformanceController.GetMyEvaluationDetail)
		}
	}
}