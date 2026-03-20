package models

import (
	"time"
)

// Patient represents the person receiving care
type Patient struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Gender    string    `json:"gender"`
	CreatedAt time.Time `json:"created_at"`
}

// MedicalRecord stores the doctor's raw input
type MedicalRecord struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	PatientID   uint         `json:"patient_id"`
	RawNote     string       `json:"raw_note"`
	ExtraCharge float64      `json:"extra_charge" gorm:"default:0"`
	TotalBill   float64      `json:"total_bill"`
	CreatedAt   time.Time    `json:"created_at"`
	ParsedItems []ParsedItem `json:"items" gorm:"foreignKey:RecordID"`
}

// ParsedItem stores individual drugs or tests extracted by AI
type ParsedItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	RecordID  uint    `json:"record_id"`
	Category  string  `json:"category"` // 'Drug', 'Lab Test', 'Observation'
	ItemName  string  `json:"item_name"`
	Dosage    string  `json:"dosage"`
	Price     float64 `json:"price"`
}

// PriceList is our reference table for billing
type PriceList struct {
	ItemName string  `json:"item_name" gorm:"primaryKey"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}