package models

type PerformanceIndicator struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"size:200" json:"name"`
	Description   string     `gorm:"type:text" json:"description"`
	IndicatorType string     `gorm:"size:20" json:"indicator_type"` // umum, spesifik
	Weight        float64    `json:"weight"`
	DivisionID    *uint      `json:"division_id"`
	
	Division  Division   `gorm:"foreignKey:DivisionID" json:"division"`
	Divisions []Division `gorm:"many2many:indicator_divisions;" json:"divisions"`
}