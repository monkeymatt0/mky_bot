package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderAudit rappresenta un record di audit per tracciare le modifiche agli ordini
type OrderAudit struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID   string    `gorm:"type:varchar(50);not null;index:idx_audit_order_id" json:"order_id"`
	FieldName string    `gorm:"type:varchar(50);not null;index:idx_field_name" json:"field_name"`
	OldValue  *string   `gorm:"type:text" json:"old_value"`
	NewValue  *string   `gorm:"type:text" json:"new_value"`
	ChangedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index:idx_changed_at" json:"changed_at"`
	ChangedBy string    `gorm:"type:varchar(100);default:'system'" json:"changed_by"`

	// Relazione con Order (opzionale per query) - senza foreign key per evitare dipendenza circolare
	Order *Order `gorm:"-" json:"order,omitempty"`
}

// TableName specifica il nome della tabella per GORM
func (OrderAudit) TableName() string {
	return "order_audit"
}

// BeforeCreate hook per validazioni prima della creazione
func (oa *OrderAudit) BeforeCreate(tx *gorm.DB) error {
	if oa.OrderID == "" || oa.FieldName == "" {
		return gorm.ErrInvalidData
	}

	if oa.ChangedBy == "" {
		oa.ChangedBy = "system"
	}

	return nil
}

// IsSignificantChange verifica se la modifica è significativa
func (oa *OrderAudit) IsSignificantChange() bool {
	if oa.OldValue == nil && oa.NewValue == nil {
		return false
	}

	if oa.OldValue == nil || oa.NewValue == nil {
		return true
	}

	return *oa.OldValue != *oa.NewValue
}

// GetOldValue restituisce il vecchio valore come stringa
func (oa *OrderAudit) GetOldValue() string {
	if oa.OldValue == nil {
		return ""
	}
	return *oa.OldValue
}

// GetNewValue restituisce il nuovo valore come stringa
func (oa *OrderAudit) GetNewValue() string {
	if oa.NewValue == nil {
		return ""
	}
	return *oa.NewValue
}

// SetOldValue imposta il vecchio valore
func (oa *OrderAudit) SetOldValue(value string) {
	oa.OldValue = &value
}

// SetNewValue imposta il nuovo valore
func (oa *OrderAudit) SetNewValue(value string) {
	oa.NewValue = &value
}

// String restituisce una rappresentazione stringa dell'audit
func (oa *OrderAudit) String() string {
	return oa.OrderID + " - " + oa.FieldName
}
