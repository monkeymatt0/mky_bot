package repositories

import (
	"context"
	"cross-exchange-arbitrage/models"

	"gorm.io/gorm"
)

// orderAuditRepository implementa OrderAuditRepository
type orderAuditRepository struct {
	db *gorm.DB
}

// NewOrderAuditRepository crea una nuova istanza di OrderAuditRepository
func NewOrderAuditRepository(db *gorm.DB) OrderAuditRepository {
	return &orderAuditRepository{db: db}
}

// Create crea un nuovo record di audit
func (r *orderAuditRepository) Create(ctx context.Context, audit *models.OrderAudit) error {
	return r.db.WithContext(ctx).Create(audit).Error
}

// GetByOrderID recupera tutti i record di audit per un ordine
func (r *orderAuditRepository) GetByOrderID(ctx context.Context, orderID string, limit, offset int) ([]*models.OrderAudit, error) {
	var audits []*models.OrderAudit
	query := r.db.WithContext(ctx).Where("order_id = ?", orderID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("changed_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

// GetByFieldName recupera record di audit per campo
func (r *orderAuditRepository) GetByFieldName(ctx context.Context, fieldName string, limit, offset int) ([]*models.OrderAudit, error) {
	var audits []*models.OrderAudit
	query := r.db.WithContext(ctx).Where("field_name = ?", fieldName)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("changed_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

// GetByDateRange recupera record di audit in un range di date
func (r *orderAuditRepository) GetByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*models.OrderAudit, error) {
	var audits []*models.OrderAudit
	query := r.db.WithContext(ctx).Where("changed_at >= ? AND changed_at <= ?", startDate, endDate)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("changed_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

// GetByOrderIDAndField recupera record di audit per ordine e campo specifico
func (r *orderAuditRepository) GetByOrderIDAndField(ctx context.Context, orderID, fieldName string) ([]*models.OrderAudit, error) {
	var audits []*models.OrderAudit
	err := r.db.WithContext(ctx).Where("order_id = ? AND field_name = ?", orderID, fieldName).
		Order("changed_at DESC").Find(&audits).Error
	if err != nil {
		return nil, err
	}
	return audits, nil
}

// CountByOrderID conta i record di audit per un ordine
func (r *orderAuditRepository) CountByOrderID(ctx context.Context, orderID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OrderAudit{}).Where("order_id = ?", orderID).Count(&count).Error
	return count, err
}

// GetLatestByOrderID recupera l'ultimo record di audit per un ordine
func (r *orderAuditRepository) GetLatestByOrderID(ctx context.Context, orderID string) (*models.OrderAudit, error) {
	var audit models.OrderAudit
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).
		Order("changed_at DESC").First(&audit).Error
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

// DeleteOldRecords elimina record di audit più vecchi di una data
func (r *orderAuditRepository) DeleteOldRecords(ctx context.Context, beforeDate string) error {
	return r.db.WithContext(ctx).Where("changed_at < ?", beforeDate).Delete(&models.OrderAudit{}).Error
}
