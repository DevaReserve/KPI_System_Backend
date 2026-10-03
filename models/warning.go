package models

import "time"

type Warning struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EmployeeID  uint      `json:"employee_id"`
	Employee    Employee  `gorm:"foreignKey:EmployeeID" json:"employee"` // Relasi ke Pegawai yg kena SP
	
	IssuedByID  uint      `json:"issued_by_id"`
	IssuedBy    Employee  `gorm:"foreignKey:IssuedByID" json:"issuer"`   // Relasi ke Manager/Admin yg menerbitkan
	
	Level       string    `json:"level"`       // "SP1", "SP2", "SP3", "TEGURAN"
	Reason      string    `json:"reason"`      // Alasan/Pelanggaran
	Description string    `json:"description"` // Detail tambahan/Saran perbaikan
	IssuedAt    time.Time `json:"issued_at"`
	
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}