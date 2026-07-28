package models

import "time"

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Title     string    `gorm:"type:varchar(100);not null" json:"title"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	Type      string    `gorm:"type:varchar(50)" json:"type"` // evaluation, appeal
	CreatedAt time.Time `json:"created_at"`
}
