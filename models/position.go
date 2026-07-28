package models

import "time"

type Position struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	DivisionID  uint      `json:"division_id"`
	Division    Division  `gorm:"foreignKey:DivisionID" json:"division,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}