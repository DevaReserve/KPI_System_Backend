package models

import (
	"time"
)

type Employee struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	NIP                string    `gorm:"column:n_ip;size:20;uniqueIndex" json:"nip"`
	Name               string    `gorm:"size:100" json:"name"`
	Email              string    `gorm:"size:100;uniqueIndex" json:"email"`
	DivisionID         uint      `json:"division_id"`
	Position           string    `gorm:"size:100" json:"position"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	JoinDate           time.Time `json:"join_date"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	ProfilePictureURL  string    `gorm:"type:varchar(255)" json:"profile_picture_url"`

	Division       Division `gorm:"foreignKey:DivisionID" json:"division,omitempty"`
	DivisionName   string   `gorm:"-" json:"division_name"`
	SupervisorName string   `gorm:"-" json:"supervisor_name"`
}

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

	Username    string `json:"username"`
	Role        string `json:"role"`
	IsExecutive bool   `json:"is_executive"`

	ProfilePictureURL string `json:"profile_picture_url"` // <--- TAMBAHKAN INI
}
