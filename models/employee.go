package models

import (
	"time"
)

type Employee struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	NIP                string    `gorm:"size:20;uniqueIndex" json:"nip"` // Ingat, ini jadi n_ip
	Name               string    `gorm:"size:100" json:"name"`
	Email              string    `gorm:"size:100;uniqueIndex" json:"email"`
	DivisionID         uint      `json:"division_id"`
	Position           string    `gorm:"size:100" json:"position"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	JoinDate           time.Time `json:"join_date"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// --- INI PENAMBAHANNYA ---
	// Tambahkan relasi ini agar GORM Preload("Employee.Division") berfungsi
	Division Division `gorm:"foreignKey:DivisionID" json:"division,omitempty"`
	// --- AKHIR PENAMBAHAN ---

	// Remove circular references - use simple fields for JSON
	DivisionName   string `gorm:"-" json:"division_name"`
	SupervisorName string `gorm:"-" json:"supervisor_name"`
}

// Separate struct for detailed employee response
type EmployeeDetail struct {
	ID                 uint      `json:"id"`
	NIP                string    `json:"nip"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	DivisionID         uint      `json:"division_id"`
	DivisionName       string    `json:"division_name"`
	Position           string    `json:"position"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	SupervisorName     string    `json:"supervisor_name"`
	IsActive           bool      `json:"is_active"`
	JoinDate           time.Time `json:"join_date"`
}