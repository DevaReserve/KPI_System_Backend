package models

import "time"

// KPITarget adalah target nilai yang ditetapkan manager untuk pegawai
// di awal periode evaluasi untuk setiap indikator KPI
type KPITarget struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EmployeeID  uint      `gorm:"not null;index" json:"employee_id"`
	PeriodID    uint      `gorm:"not null;index" json:"period_id"`
	IndicatorID uint      `gorm:"not null" json:"indicator_id"`
	TargetScore int       `gorm:"default:3" json:"target_score"` // 1-5 (sama skala dengan penilaian)
	Notes       string    `gorm:"type:text" json:"notes"`
	SetByID     uint      `json:"set_by_id"` // Employee ID dari manager yang set target
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations (tidak disimpan di DB)
	Employee  Employee             `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Period    EvaluationPeriod     `gorm:"foreignKey:PeriodID" json:"period,omitempty"`
	Indicator PerformanceIndicator `gorm:"foreignKey:IndicatorID" json:"indicator,omitempty"`
}
