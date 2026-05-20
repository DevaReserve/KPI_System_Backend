package controllers

import (
	"KPI_System_Backend/db_var"
	"KPI_System_Backend/helper"
	"KPI_System_Backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EmployeeController struct {
	DB *gorm.DB
}

// NewEmployeeController adalah "constructor"
func NewEmployeeController(db *gorm.DB) *EmployeeController {
	return &EmployeeController{DB: db}
}

type EmployeeCreateRequest struct {
	// Data untuk models.Employee
	NIP                string    `json:"nip" binding:"required"`
	Name               string    `json:"name" binding:"required"`
	Email              string    `json:"email" binding:"required,email"`
	DivisionID         uint      `json:"division_id" binding:"required"`
	Position           string    `json:"position" binding:"required"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	JoinDate           time.Time `json:"join_date" binding:"required"`

	// Data untuk models.User
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	IsExecutive bool `json:"is_executive"`
}

// Struct baru untuk Update
type EmployeeUpdateRequest struct {
	// Data Employee yang bisa diubah
	Name               string    `json:"name" binding:"required"`
	Email              string    `json:"email" binding:"required,email"`
	DivisionID         uint      `json:"division_id" binding:"required"`
	Position           string    `json:"position" binding:"required"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	JoinDate           time.Time `json:"join_date" binding:"required"`

	// Data User yang bisa diubah
	Username string `json:"username" binding:"required"`
	Role     string `json:"role" binding:"required"` // "admin", "manager", atau "employee"
	IsActive bool   `json:"is_active"`

	IsExecutive bool `json:"is_executive"`
}

// --- CRUD Functions ---

// CreateEmployee: Membuat pegawai baru (dan akun user-nya)
// @Route: POST /api/admin/employees
func (ec *EmployeeController) CreateEmployee(c *gin.Context) {
	var req EmployeeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}
	if req.Role != db_var.RoleAdmin && req.Role != db_var.RoleManager && req.Role != db_var.RoleEmployee {
		Response(c, http.StatusBadRequest, "Role tidak valid. Gunakan 'admin', 'manager', atau 'employee'", nil)
		return
	}
	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memproses password", nil)
		return
	}
	employee := models.Employee{
		NIP:                req.NIP,
		Name:               req.Name,
		Email:              req.Email,
		DivisionID:         req.DivisionID,
		Position:           req.Position,
		DirectSupervisorID: req.DirectSupervisorID,
		JoinDate:           req.JoinDate,
		IsActive:           true,
	}
	user := models.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Role:         req.Role,
		IsActive:     true,
		IsExecutive:  req.IsExecutive,
	}
	tx := ec.DB.Begin()
	if tx.Error != nil {
		Response(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	if err := tx.Create(&employee).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusConflict, "Gagal membuat pegawai, NIP atau Email mungkin sudah terdaftar", nil)
		return
	}
	user.EmployeeID = employee.ID
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusConflict, "Gagal membuat user, Username mungkin sudah terdaftar", nil)
		return
	}
	if err := tx.Commit().Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan data", nil)
		return
	}
	user.PasswordHash = ""
	response := gin.H{"employee": employee, "user": user}
	Response(c, http.StatusCreated, db_var.MsgEmployeeCreated, response)
}

// GetAllEmployees: Mendapatkan semua pegawai
// @Route: GET /api/admin/employees
func (ec *EmployeeController) GetAllEmployees(c *gin.Context) {
	var employeeDetails []models.EmployeeDetail
	
    // Query ini menggabungkan data dari 3 tabel: employees, divisions, dan users
	query := ec.DB.Model(&models.Employee{}).
		Select("employees.*, divisions.name as division_name, supervisors.name as supervisor_name, users.username, users.role, users.is_executive").
		Joins("left join divisions on divisions.id = employees.division_id").
		Joins("left join employees as supervisors on supervisors.id = employees.direct_supervisor_id").
		Joins("left join users on users.employee_id = employees.id") 

	if err := query.Scan(&employeeDetails).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mengambil data pegawai", nil)
		return
	}
	Response(c, http.StatusOK, "Data semua pegawai berhasil diambil", employeeDetails)
}
// GetEmployee: Mendapatkan detail satu pegawai
// @Route: GET /api/admin/employees/:id
func (ec *EmployeeController) GetEmployee(c *gin.Context) {
	id := c.Param("id")
	var employeeDetail models.EmployeeDetail
	query := ec.DB.Model(&models.Employee{}).
		Select("employees.*, divisions.name as division_name, supervisors.name as supervisor_name").
		Joins("left join divisions on divisions.id = employees.division_id").
		Joins("left join employees as supervisors on supervisors.id = employees.direct_supervisor_id")
	if err := query.Where("employees.id = ?", id).First(&employeeDetail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mengambil data pegawai", nil)
		return
	}
	Response(c, http.StatusOK, "Data pegawai berhasil diambil", employeeDetail)
}

// UpdateEmployee: Memperbarui data pegawai
// @Route: PUT /api/admin/employees/:id
func (ec *EmployeeController) UpdateEmployee(c *gin.Context) {
	id := c.Param("id")
	var req EmployeeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Response(c, http.StatusBadRequest, "Format request tidak valid", err.Error())
		return
	}
	if req.Role != db_var.RoleAdmin && req.Role != db_var.RoleManager && req.Role != db_var.RoleEmployee {
		Response(c, http.StatusBadRequest, "Role tidak valid. Gunakan 'admin', 'manager', atau 'employee'", nil)
		return
	}
	tx := ec.DB.Begin()
	if tx.Error != nil {
		Response(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}
	var employee models.Employee
	if err := tx.First(&employee, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mencari data pegawai", nil)
		return
	}
	var user models.User
	if err := tx.Where("employee_id = ?", employee.ID).First(&user).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusNotFound, "Data user untuk pegawai ini tidak ditemukan", nil)
		return
	}
	employee.Name = req.Name
	employee.Email = req.Email
	employee.DivisionID = req.DivisionID
	employee.Position = req.Position
	employee.DirectSupervisorID = req.DirectSupervisorID
	employee.JoinDate = req.JoinDate
	employee.IsActive = req.IsActive
	user.Username = req.Username
	user.Role = req.Role
	user.IsActive = req.IsActive
	user.IsExecutive = req.IsExecutive
	if err := tx.Save(&employee).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusConflict, "Gagal update pegawai, Email mungkin duplikat", nil)
		return
	}
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusConflict, "Gagal update user, Username mungkin duplikat", nil)
		return
	}
	if err := tx.Commit().Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan perubahan", nil)
		return
	}

	// --- LOG ACTIVITY (Baris Baru) ---
	// Ambil ID User yang sedang login (Pelaku Edit)
	actorID, _ := c.Get("userID")
	if idUint, ok := actorID.(uint); ok {
		helper.LogActivity(ec.DB, idUint, "UPDATE_EMPLOYEE", "Mengupdate data pegawai: "+employee.Name, c.ClientIP())
	}

	user.PasswordHash = ""
	response := gin.H{"employee": employee, "user": user}
	Response(c, http.StatusOK, db_var.MsgEmployeeUpdated, response)
}

// DeleteEmployee: Menonaktifkan pegawai (soft delete)
// @Route: DELETE /api/admin/employees/:id
func (ec *EmployeeController) DeleteEmployee(c *gin.Context) {
	// 1. Ambil ID dari URL
	id := c.Param("id")

	// 2. Mulai Transaksi
	tx := ec.DB.Begin()
	if tx.Error != nil {
		Response(c, http.StatusInternalServerError, "Gagal memulai transaksi", nil)
		return
	}

	// 3. Cari data Pegawai (Employee)
	var employee models.Employee
	if err := tx.First(&employee, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			Response(c, http.StatusNotFound, "Pegawai tidak ditemukan", nil)
			return
		}
		Response(c, http.StatusInternalServerError, "Gagal mencari data pegawai", nil)
		return
	}

	// 4. Cari data User yang terhubung
	var user models.User
	if err := tx.Where("employee_id = ?", employee.ID).First(&user).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusNotFound, "Data user untuk pegawai ini tidak ditemukan", nil)
		return
	}

	// 5. Lakukan "Soft Delete"
	employee.IsActive = false
	user.IsActive = false

	// 6. Simpan perubahan Employee
	if err := tx.Save(&employee).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menonaktifkan pegawai", nil)
		return
	}

	// 7. Simpan perubahan User
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		Response(c, http.StatusInternalServerError, "Gagal menonaktifkan user", nil)
		return
	}

	// 8. Commit Transaksi
	if err := tx.Commit().Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal menyimpan perubahan", nil)
		return
	}

	Response(c, http.StatusOK, "Pegawai berhasil dinonaktifkan", nil)
}

func (ec *EmployeeController) ResetPassword(c *gin.Context) {
	id := c.Param("id")

	// 1. Cari User berdasarkan Employee ID
	var user models.User
	if err := ec.DB.Where("employee_id = ?", id).First(&user).Error; err != nil {
		Response(c, http.StatusNotFound, "Akun pengguna tidak ditemukan untuk pegawai ini", nil)
		return
	}

	// 2. Hash Password Default ('cakra123')
	// Pastikan Anda sudah import "KPI_System_Backend/helper"
	newHash, err := helper.HashPassword("cakra123") 
	if err != nil {
		Response(c, http.StatusInternalServerError, "Gagal memproses password", nil)
		return
	}

	// 3. Update Database
	if err := ec.DB.Model(&user).Update("password_hash", newHash).Error; err != nil {
		Response(c, http.StatusInternalServerError, "Gagal mereset password", nil)
		return
	}

	Response(c, http.StatusOK, "Password berhasil direset menjadi 'cakra123'", nil)
}