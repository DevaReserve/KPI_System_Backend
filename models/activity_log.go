package models

import (
	"time"
)

type ActivityLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	IpAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}