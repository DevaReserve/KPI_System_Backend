package models

type EvaluationScore struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	EvaluationID   uint    `json:"evaluation_id"`
	IndicatorID    uint    `json:"indicator_id"`
	Score          int     `json:"score"` // 1-5
	ConvertedScore int     `json:"converted_score"` // 20,40,60,80,100
	Notes          string  `gorm:"type:text" json:"notes"`
	
	Indicator PerformanceIndicator `gorm:"foreignKey:IndicatorID" json:"indicator"`
}