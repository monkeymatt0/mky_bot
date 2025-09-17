package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// repositoryManager implementa RepositoryManager
type repositoryManager struct {
	db              *gorm.DB
	orderStatusRepo OrderStatusRepository
	orderRepo       OrderRepository
	orderAuditRepo  OrderAuditRepository
}

// NewRepositoryManager crea una nuova istanza di RepositoryManager
func NewRepositoryManager(db *gorm.DB) RepositoryManager {
	return &repositoryManager{
		db:              db,
		orderStatusRepo: NewOrderStatusRepository(db),
		orderRepo:       NewOrderRepository(db),
		orderAuditRepo:  NewOrderAuditRepository(db),
	}
}

// OrderStatus restituisce il repository per gli stati ordine
func (rm *repositoryManager) OrderStatus() OrderStatusRepository {
	return rm.orderStatusRepo
}

// Order restituisce il repository per gli ordini
func (rm *repositoryManager) Order() OrderRepository {
	return rm.orderRepo
}

// OrderAudit restituisce il repository per l'audit trail
func (rm *repositoryManager) OrderAudit() OrderAuditRepository {
	return rm.orderAuditRepo
}

// BeginTransaction inizia una transazione
func (rm *repositoryManager) BeginTransaction(ctx context.Context) (*gorm.DB, error) {
	if rm.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	if err := rm.db.WithContext(ctx).Error; err != nil {
		return nil, err
	}
	return rm.db.WithContext(ctx).Begin(), nil
}

// CommitTransaction committa una transazione
func (rm *repositoryManager) CommitTransaction(tx *gorm.DB) error {
	return tx.Commit().Error
}

// RollbackTransaction fa rollback di una transazione
func (rm *repositoryManager) RollbackTransaction(tx *gorm.DB) error {
	return tx.Rollback().Error
}
