package repositories

import (
	"context"
	"cross-exchange-arbitrage/models"

	"gorm.io/gorm"
)

// orderStatusRepository implementa OrderStatusRepository
type orderStatusRepository struct {
	db *gorm.DB
}

// NewOrderStatusRepository crea una nuova istanza di OrderStatusRepository
func NewOrderStatusRepository(db *gorm.DB) OrderStatusRepository {
	return &orderStatusRepository{db: db}
}

// Create crea un nuovo stato ordine
func (r *orderStatusRepository) Create(ctx context.Context, orderStatus *models.OrderStatusEntity) error {
	return r.db.WithContext(ctx).Create(orderStatus).Error
}

// GetByID recupera uno stato ordine per ID
func (r *orderStatusRepository) GetByID(ctx context.Context, id uint) (*models.OrderStatusEntity, error) {
	var orderStatus models.OrderStatusEntity
	err := r.db.WithContext(ctx).First(&orderStatus, id).Error
	if err != nil {
		return nil, err
	}
	return &orderStatus, nil
}

// GetByStatusName recupera uno stato ordine per nome
func (r *orderStatusRepository) GetByStatusName(ctx context.Context, statusName string) (*models.OrderStatusEntity, error) {
	var orderStatus models.OrderStatusEntity
	err := r.db.WithContext(ctx).Where("status_name = ?", statusName).First(&orderStatus).Error
	if err != nil {
		return nil, err
	}
	return &orderStatus, nil
}

// GetAll recupera tutti gli stati ordine
func (r *orderStatusRepository) GetAll(ctx context.Context) ([]*models.OrderStatusEntity, error) {
	var orderStatuses []*models.OrderStatusEntity
	err := r.db.WithContext(ctx).Find(&orderStatuses).Error
	if err != nil {
		return nil, err
	}
	return orderStatuses, nil
}

// GetActive recupera solo gli stati attivi
func (r *orderStatusRepository) GetActive(ctx context.Context) ([]*models.OrderStatusEntity, error) {
	var orderStatuses []*models.OrderStatusEntity
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&orderStatuses).Error
	if err != nil {
		return nil, err
	}
	return orderStatuses, nil
}

// Update aggiorna uno stato ordine esistente
func (r *orderStatusRepository) Update(ctx context.Context, orderStatus *models.OrderStatusEntity) error {
	return r.db.WithContext(ctx).Save(orderStatus).Error
}

// Delete elimina uno stato ordine (soft delete)
func (r *orderStatusRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.OrderStatusEntity{}, id).Error
}

// Exists verifica se uno stato ordine esiste
func (r *orderStatusRepository) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.OrderStatusEntity{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
