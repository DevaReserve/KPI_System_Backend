package routes

import (
	"KPI_System_Backend/controllers"
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Init Controllers
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
	activityCtrl := controllers.NewActivityController(db)
	warningCtrl := controllers.NewWarningController(db)
	
	// Controller Baru untuk Upload
	uploadCtrl := controllers.NewUploadController(db)
	achievementsCtrl := controllers.NewAchievementController(db)

	// ---------------------------------------------------------
	// PUBLIC ROUTES
	// ---------------------------------------------------------
	public := router.Group("/api")
	public.Use(middleware.CORSMiddleware()) 
	{
		public.POST("/auth/login", authController.Login)
		public.GET("/ping", pingController.Ping)
	}

	// ---------------------------------------------------------
	// PROTECTED ROUTES (Butuh Token)
	// ---------------------------------------------------------
	protected := router.Group("/api")
	protected.Use(middleware.CORSMiddleware())
	protected.Use(middleware.AuthMiddleware())
	{
		// Common Routes (Bisa diakses semua user login)
		protected.GET("/auth/profile", authController.GetProfile)
		protected.PUT("/auth/change-password", authController.ChangePassword)

		// ---------------------------------------------------------
		// ADMIN ROUTES (Hanya Admin)
		// ---------------------------------------------------------
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleAdmin))
		{
			// Divisions
			adminRoutes.POST("/divisions", divisionController.CreateDivision)
			adminRoutes.GET("/divisions", divisionController.GetAllDivisions)
			adminRoutes.GET("/divisions/:id", divisionController.GetDivision)
			adminRoutes.PUT("/divisions/:id", divisionController.UpdateDivision)
			adminRoutes.DELETE("/divisions/:id", divisionController.DeleteDivision)

			// Positions
			adminRoutes.POST("/positions", positionController.CreatePosition)
			adminRoutes.GET("/positions", positionController.GetAllPositions)
			adminRoutes.PUT("/positions/:id", positionController.UpdatePosition)
			adminRoutes.DELETE("/positions/:id", positionController.DeletePosition)

			// Employees
			adminRoutes.POST("/employees", employeeController.CreateEmployee)
			adminRoutes.GET("/employees", employeeController.GetAllEmployees)
			adminRoutes.GET("/employees/:id", employeeController.GetEmployee)
			adminRoutes.PUT("/employees/:id", employeeController.UpdateEmployee)
			adminRoutes.DELETE("/employees/:id", employeeController.DeleteEmployee)
			adminRoutes.PATCH("/employees/:id/reset-password", employeeController.ResetPassword)

			// Indicators
			adminRoutes.POST("/indicators", indicatorController.CreateIndicator)
			adminRoutes.GET("/indicators", indicatorController.GetAllIndicators)
			adminRoutes.GET("/indicators/:id", indicatorController.GetIndicator)
			adminRoutes.PUT("/indicators/:id", indicatorController.UpdateIndicator)
			adminRoutes.DELETE("/indicators/:id", indicatorController.DeleteIndicator)

			// Periods
			adminRoutes.POST("/periods", periodController.CreatePeriod)
			adminRoutes.GET("/periods", periodController.GetAllPeriods)
			adminRoutes.GET("/periods/:id", periodController.GetPeriod)
			adminRoutes.PUT("/periods/:id", periodController.UpdatePeriod)
			adminRoutes.DELETE("/periods/:id", periodController.DeletePeriod)
			adminRoutes.PATCH("/periods/:id/activate", periodController.SetActivePeriod)

			// Reports
			adminRoutes.GET("/reports/evaluations", reportController.GetEvaluationReport)

			// Achevement
			adminRoutes.GET("/employees/:id/achievements", achievementsCtrl.GetEmployeeAchievements)

			// Activity Logs
			adminRoutes.GET("/activity-logs", activityCtrl.GetAllLogs)

			// Warnings
			adminRoutes.POST("/warnings", warningCtrl.CreateWarning)          			// Terbitkan SP
            adminRoutes.DELETE("/warnings/:id", warningCtrl.DeleteWarning)    			// Hapus SP
            adminRoutes.GET("/employees/:id/warnings", warningCtrl.GetEmployeeWarnings) // Lihat SP Pegawai
		}

		// ---------------------------------------------------------
		// MANAGER ROUTES (Admin & Manager)
		// ---------------------------------------------------------
		managerRoutes := protected.Group("/manager")
		managerRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleManager, db_var.RoleAdmin))
		{
			managerRoutes.GET("/my-team", managerController.GetMyTeam)
			managerRoutes.GET("/team-status", managerController.GetTeamEvaluationStatus)
			managerRoutes.POST("/evaluations/start", managerController.StartEvaluation)
			managerRoutes.GET("/evaluations/:id", managerController.GetEvaluationDetail)
			managerRoutes.PUT("/evaluations/:id/submit", managerController.SubmitEvaluation)

			// Achievement
			managerRoutes.GET("/employees/:id/achievements", achievementsCtrl.GetEmployeeAchievements)

			// Warnings
			managerRoutes.POST("/warnings", warningCtrl.CreateWarning)
            managerRoutes.GET("/employees/:id/warnings", warningCtrl.GetEmployeeWarnings)
		}

		// ---------------------------------------------------------
		// EMPLOYEE ROUTES (Semua Role bisa akses riwayat sendiri)
		// ---------------------------------------------------------
		employeeRoutes := protected.Group("/employee")
        employeeRoutes.Use(middleware.RoleCheckMiddleware(db_var.RoleEmployee, db_var.RoleManager, db_var.RoleAdmin))
        {
            employeeRoutes.GET("/history", myPerformanceController.GetMyPerformanceHistory)
            employeeRoutes.GET("/latest", myPerformanceController.GetMyLatestPerformance)
            employeeRoutes.GET("/evaluations/:id", myPerformanceController.GetMyEvaluationDetail)
            
            // Upload Foto Profil
            employeeRoutes.POST("/upload-avatar", uploadCtrl.UploadProfilePicture)

            // --- 2. TAMBAHKAN ROUTE PRESTASI DISINI ---
            employeeRoutes.GET("/achievements", achievementsCtrl.GetMyAchievements)
            employeeRoutes.POST("/achievements", achievementsCtrl.CreateAchievement)
            employeeRoutes.DELETE("/achievements/:id", achievementsCtrl.DeleteAchievement)
			employeeRoutes.PUT("/achievements/:id", achievementsCtrl.UpdateAchievement)

			// Warnings
			employeeRoutes.GET("/warnings", warningCtrl.GetMyWarnings)
        }
	}
}