package models

import (
	"time"
)

type EmployeeAchievement struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EmployeeID  uint      `json:"employee_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	FileURL     string    `json:"file_url"`
	CreatedAt   time.Time `json:"created_at"`
}