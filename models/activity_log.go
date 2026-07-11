package models

import (
	"time"
)

type ActivityLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      *uint     `gorm:"index;default:null" json:"user_id,omitempty"`
	User        *User     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user,omitempty"`
	Username    string    `gorm:"size:100" json:"username,omitempty"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	IpAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}
