package models

import (
	"time"
)

type Evaluation struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	EmployeeID  uint       `json:"employee_id"`
	EvaluatorID uint       `json:"evaluator_id"`
	PeriodID    uint       `json:"period_id"`
	TotalScore  float64    `json:"total_score"`
	Feedback    string     `gorm:"type:text" json:"feedback"`
	Status      string     `gorm:"size:20;default:draft" json:"status"`
	SubmittedAt *time.Time `json:"submitted_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	AppealReason string `gorm:"type:text" json:"appeal_reason"`
	EvidenceURL  string `gorm:"type:varchar(255)" json:"evidence_url"`

	// Simple fields for JSON response
	EmployeeName  string `gorm:"-" json:"employee_name"`
	EvaluatorName string `gorm:"-" json:"evaluator_name"`
	PeriodName    string `gorm:"-" json:"period_name"`
	DivisionName  string `gorm:"-" json:"division_name"`
}

// Detailed evaluation response
type EvaluationDetail struct {
	ID            uint       `json:"id"`
	EmployeeID    uint       `json:"employee_id"`
	EmployeeName  string     `json:"employee_name"`
	EvaluatorID   uint       `json:"evaluator_id"`
	EvaluatorName string     `json:"evaluator_name"`
	PeriodID      uint       `json:"period_id"`
	PeriodName    string     `json:"period_name"`
	TotalScore    float64    `json:"total_score"`
	Feedback      string     `json:"feedback"`
	Status        string     `json:"status"`
	SubmittedAt   *time.Time `json:"submitted_at"`

	AppealReason string `json:"appeal_reason"`
	EvidenceURL  string `json:"evidence_url"`

	Scores []EvaluationScore `json:"scores"`
}
