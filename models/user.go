package models

import (
    "time"
)

type User struct {
    ID           uint      `gorm:"primaryKey" json:"id"`
    EmployeeID   uint      `gorm:"uniqueIndex" json:"employee_id"`
    Username     string    `gorm:"size:50;uniqueIndex" json:"username"`
    PasswordHash string    `gorm:"size:255" json:"-"`
    Role         string    `gorm:"size:20" json:"role"` // admin, manager, employee
    
    // TAMBAHAN BARU: Penanda khusus untuk hak akses CEO
    IsExecutive  bool      `gorm:"default:false" json:"is_executive"` 
    
    IsActive     bool      `gorm:"default:true" json:"is_active"`
    LastLogin    time.Time `json:"last_login"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    
    Employee Employee `gorm:"foreignKey:EmployeeID" json:"employee"`
}

type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}