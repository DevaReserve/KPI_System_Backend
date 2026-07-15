package models

import (
	"time"
)

type Employee struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	NIP                string    `gorm:"column:n_ip;size:20;uniqueIndex" json:"nip"`
	Name               string    `gorm:"size:100" json:"name"`
	Email              string    `gorm:"size:100;uniqueIndex" json:"email"`
	DivisionID         uint      `json:"division_id"`
	Position           string    `gorm:"size:100" json:"position"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	JoinDate           time.Time `json:"join_date"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	ProfilePictureURL  string    `gorm:"type:varchar(255)" json:"profile_picture_url"`
	Phone              string    `gorm:"size:50" json:"phone"`
	Bio                string    `gorm:"type:text" json:"bio"`
	SocialMedia        string    `gorm:"size:255" json:"social_media"`
	
	Address               string `gorm:"type:text" json:"address"`
	BirthPlace            string `gorm:"size:100" json:"birth_place"`
	BirthDate             string `gorm:"size:50" json:"birth_date"`
	Gender                string `gorm:"size:20" json:"gender"`
	Education             string `gorm:"size:150" json:"education"`
	EmergencyContactName  string `gorm:"size:100" json:"emergency_contact_name"`
	EmergencyContactPhone string `gorm:"size:50" json:"emergency_contact_phone"`
	
	PhoneOTP          string     `gorm:"size:10" json:"-"`
	PhoneOTPExpiredAt *time.Time `json:"-"`
	IsPhoneVerified   bool       `gorm:"default:false" json:"is_phone_verified"`

	Division       Division `gorm:"foreignKey:DivisionID" json:"division,omitempty"`
	DivisionName   string   `gorm:"-" json:"division_name"`
	SupervisorName string   `gorm:"-" json:"supervisor_name"`
}

type EmployeeDetail struct {
	ID                 uint      `json:"id"`
	NIP                string    `json:"nip"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	DivisionID         uint      `json:"division_id"`
	DivisionName       string    `json:"division_name"`
	Position           string    `json:"position"`
	DirectSupervisorID *uint     `json:"direct_supervisor_id"`
	SupervisorName     string    `json:"supervisor_name"`
	IsActive           bool      `json:"is_active"`
	JoinDate           time.Time `json:"join_date"`
	ProfilePictureURL  string    `json:"profile_picture_url"`
	Phone              string    `json:"phone"`
	Bio                string    `json:"bio"`
	SocialMedia        string    `json:"social_media"`
	
	Address               string `json:"address"`
	BirthPlace            string `json:"birth_place"`
	BirthDate             string `json:"birth_date"`
	Gender                string `json:"gender"`
	Education             string `json:"education"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	
	IsPhoneVerified    bool      `json:"is_phone_verified"`
	Username           string    `json:"username"`
	Role               string    `json:"role"`
	IsExecutive        bool      `json:"is_executive"`
}
