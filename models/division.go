package models

type Division struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	Name            string `gorm:"size:100;uniqueIndex" json:"name"`
	Description     string `gorm:"type:text" json:"description"`
	ManagerID       uint   `json:"manager_id"`
	
	// Remove circular reference - use simple struct for JSON
	ManagerName string `gorm:"-" json:"manager_name"`
}

// Separate struct for detailed division response
type DivisionDetail struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ManagerID   uint       `json:"manager_id"`
	ManagerName string     `json:"manager_name"`
	EmployeeCount int      `json:"employee_count"`
}