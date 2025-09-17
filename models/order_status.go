package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatusEntity rappresenta lo stato di un ordine nel sistema
type OrderStatusEntity struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	StatusName  string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_status_name" json:"status_name"`
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"type:boolean;default:true;index:idx_is_active" json:"is_active"`
	CreatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifica il nome della tabella per GORM
func (OrderStatusEntity) TableName() string {
	return "order_statuses"
}

// BeforeCreate hook per validazioni prima della creazione
func (os *OrderStatusEntity) BeforeCreate(tx *gorm.DB) error {
	if os.StatusName == "" {
		return gorm.ErrInvalidData
	}
	return nil
}

// BeforeUpdate hook per validazioni prima dell'aggiornamento
func (os *OrderStatusEntity) BeforeUpdate(tx *gorm.DB) error {
	if os.StatusName == "" {
		return gorm.ErrInvalidData
	}
	return nil
}

// IsValid verifica se lo stato è valido
func (os *OrderStatusEntity) IsValid() bool {
	return os.StatusName != "" && os.IsActive
}

// String restituisce una rappresentazione stringa dello stato
func (os *OrderStatusEntity) String() string {
	return os.StatusName
}
