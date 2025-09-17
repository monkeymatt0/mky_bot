package services

import (
	"context"
	"cross-exchange-arbitrage/models"
	"cross-exchange-arbitrage/repositories"
	"fmt"

	"gorm.io/gorm"
)

// OrderService gestisce la logica business per gli ordini
type OrderService struct {
	repoManager repositories.RepositoryManager
}

// NewOrderService crea una nuova istanza di OrderService
func NewOrderService(repoManager repositories.RepositoryManager) *OrderService {
	return &OrderService{
		repoManager: repoManager,
	}
}

// CreateOrder crea un nuovo ordine con validazioni business
func (s *OrderService) CreateOrder(ctx context.Context, order *models.Order) error {
	// Validazioni business
	if err := s.validateOrder(order); err != nil {
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Verifica che l'ordine non esista già
	exists, err := s.repoManager.Order().Exists(ctx, order.OrderID)
	if err != nil {
		return fmt.Errorf("failed to check order existence: %w", err)
	}
	if exists {
		return fmt.Errorf("order with ID %s already exists", order.OrderID)
	}

	// Verifica che lo stato ordine esista
	status, err := s.repoManager.OrderStatus().GetByID(ctx, order.OrderStatusID)
	if err != nil {
		return fmt.Errorf("invalid order status: %w", err)
	}
	if !status.IsActive {
		return fmt.Errorf("order status %s is not active", status.StatusName)
	}

	// Crea l'ordine
	if err := s.repoManager.Order().Create(ctx, order); err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Crea record di audit per la creazione
	audit := &models.OrderAudit{
		OrderID:   order.OrderID,
		FieldName: "created",
		OldValue:  nil,
		NewValue:  func() *string { v := "Order created"; return &v }(),
		ChangedBy: "system",
	}
	if err := s.repoManager.OrderAudit().Create(ctx, audit); err != nil {
		// Log dell'errore ma non fallisce la creazione dell'ordine
		fmt.Printf("Warning: failed to create audit record: %v\n", err)
	}

	return nil
}

// UpdateOrder aggiorna un ordine esistente con audit trail
func (s *OrderService) UpdateOrder(ctx context.Context, order *models.Order) error {
	// Recupera l'ordine esistente per confronto
	existingOrder, err := s.repoManager.Order().GetByOrderID(ctx, order.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get existing order: %w", err)
	}

	// Validazioni business
	if err := s.validateOrder(order); err != nil {
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Inizia transazione per aggiornamento e audit
	tx, err := s.repoManager.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Aggiorna l'ordine
	if err := tx.Save(order).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Crea record di audit per le modifiche
	if err := s.createAuditRecords(ctx, tx, existingOrder, order); err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to create audit records: %w", err)
	}

	// Commit transazione
	if err := s.repoManager.CommitTransaction(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateOrderStatus aggiorna solo lo stato di un ordine
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, statusName string) error {
	// Verifica che lo stato esista
	status, err := s.repoManager.OrderStatus().GetByStatusName(ctx, statusName)
	if err != nil {
		return fmt.Errorf("invalid order status: %w", err)
	}

	// Recupera l'ordine esistente
	order, err := s.repoManager.Order().GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Verifica se lo stato è cambiato
	if order.OrderStatusID == status.ID {
		return nil // Nessun cambiamento necessario
	}

	// Inizia transazione
	tx, err := s.repoManager.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Aggiorna lo stato
	if err := tx.Model(&models.Order{}).Where("order_id = ?", orderID).Update("order_status_id", status.ID).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Crea record di audit
	audit := &models.OrderAudit{
		OrderID:   orderID,
		FieldName: "order_status_id",
		OldValue:  func() *string { v := fmt.Sprintf("%d", order.OrderStatusID); return &v }(),
		NewValue:  func() *string { v := fmt.Sprintf("%d", status.ID); return &v }(),
		ChangedBy: "system",
	}
	if err := tx.Create(audit).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to create audit record: %w", err)
	}

	// Commit transazione
	if err := s.repoManager.CommitTransaction(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateOrderResult aggiorna il risultato di un ordine
func (s *OrderService) UpdateOrderResult(ctx context.Context, orderID string, result models.OrderResult) error {
	// Recupera l'ordine esistente
	order, err := s.repoManager.Order().GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Verifica se il risultato è cambiato
	if order.Result == result {
		return nil // Nessun cambiamento necessario
	}

	// Inizia transazione
	tx, err := s.repoManager.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Aggiorna il risultato
	if err := tx.Model(&models.Order{}).Where("order_id = ?", orderID).Update("result", result).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to update order result: %w", err)
	}

	// Crea record di audit
	audit := &models.OrderAudit{
		OrderID:   orderID,
		FieldName: "result",
		OldValue:  func() *string { v := string(order.Result); return &v }(),
		NewValue:  func() *string { v := string(result); return &v }(),
		ChangedBy: "system",
	}
	if err := tx.Create(audit).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to create audit record: %w", err)
	}

	// Commit transazione
	if err := s.repoManager.CommitTransaction(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateOrderPnL aggiorna PnL e PnL percentage di un ordine
func (s *OrderService) UpdateOrderPnL(ctx context.Context, orderID string, currentPrice float64) error {
	// Recupera l'ordine esistente
	order, err := s.repoManager.Order().GetByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Calcola nuovo PnL
	order.CalculatePnL(currentPrice)

	// Inizia transazione
	tx, err := s.repoManager.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Aggiorna PnL
	if err := tx.Model(&models.Order{}).Where("order_id = ?", orderID).Updates(map[string]interface{}{
		"pnl":            order.PnL,
		"pnl_percentage": order.PnLPercentage,
	}).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to update order PnL: %w", err)
	}

	// Crea record di audit
	audit := &models.OrderAudit{
		OrderID:   orderID,
		FieldName: "pnl_update",
		OldValue:  func() *string { v := fmt.Sprintf("PnL: %.8f, PnL%%: %.4f", order.PnL, order.PnLPercentage); return &v }(),
		NewValue:  func() *string { v := fmt.Sprintf("PnL: %.8f, PnL%%: %.4f", order.PnL, order.PnLPercentage); return &v }(),
		ChangedBy: "system",
	}
	if err := tx.Create(audit).Error; err != nil {
		s.repoManager.RollbackTransaction(tx)
		return fmt.Errorf("failed to create audit record: %w", err)
	}

	// Commit transazione
	if err := s.repoManager.CommitTransaction(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetOrderWithAudit recupera un ordine con il suo audit trail
func (s *OrderService) GetOrderWithAudit(ctx context.Context, orderID string) (*models.Order, []*models.OrderAudit, error) {
	// Recupera l'ordine
	order, err := s.repoManager.Order().GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Recupera l'audit trail
	audits, err := s.repoManager.OrderAudit().GetByOrderID(ctx, orderID, 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audit trail: %w", err)
	}

	return order, audits, nil
}

// GetTradingStatistics recupera statistiche di trading
func (s *OrderService) GetTradingStatistics(ctx context.Context, symbol string) (*repositories.TradingStats, error) {
	stats, err := s.repoManager.Order().GetTradingStats(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading statistics: %w", err)
	}
	return stats, nil
}

// GetOrdersByResult recupera ordini per risultato
func (s *OrderService) GetOrdersByResult(ctx context.Context, result models.OrderResult) ([]*models.Order, error) {
	orders, err := s.repoManager.Order().GetByResult(ctx, result, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by result: %w", err)
	}
	return orders, nil
}

// validateOrder valida un ordine secondo le regole business
func (s *OrderService) validateOrder(order *models.Order) error {
	// Validazioni base
	if order.OrderID == "" {
		return fmt.Errorf("order ID is required")
	}
	if order.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if order.OrderPrice <= 0 {
		return fmt.Errorf("order price must be positive")
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	// Validazione side
	if order.Side != models.OrderSideTypeBuy && order.Side != models.OrderSideTypeSell {
		return fmt.Errorf("invalid order side: %s", order.Side)
	}

	// Validazione take profit
	if order.TakeProfitPrice != nil {
		if *order.TakeProfitPrice <= 0 {
			return fmt.Errorf("take profit price must be positive")
		}
		if order.Side == models.OrderSideTypeBuy && *order.TakeProfitPrice <= order.OrderPrice {
			return fmt.Errorf("take profit price must be greater than order price for Buy orders")
		}
		if order.Side == models.OrderSideTypeSell && *order.TakeProfitPrice >= order.OrderPrice {
			return fmt.Errorf("take profit price must be less than order price for Sell orders")
		}
	}

	// Validazione stop loss
	if order.StopLossPrice != nil {
		if *order.StopLossPrice <= 0 {
			return fmt.Errorf("stop loss price must be positive")
		}
		if order.Side == models.OrderSideTypeBuy && *order.StopLossPrice >= order.OrderPrice {
			return fmt.Errorf("stop loss price must be less than order price for Buy orders")
		}
		if order.Side == models.OrderSideTypeSell && *order.StopLossPrice <= order.OrderPrice {
			return fmt.Errorf("stop loss price must be greater than order price for Sell orders")
		}
	}

	return nil
}

// createAuditRecords crea record di audit per le modifiche
func (s *OrderService) createAuditRecords(ctx context.Context, tx *gorm.DB, oldOrder, newOrder *models.Order) error {
	// Audit per order_price
	if oldOrder.OrderPrice != newOrder.OrderPrice {
		audit := &models.OrderAudit{
			OrderID:   newOrder.OrderID,
			FieldName: "order_price",
			OldValue:  func() *string { v := fmt.Sprintf("%.8f", oldOrder.OrderPrice); return &v }(),
			NewValue:  func() *string { v := fmt.Sprintf("%.8f", newOrder.OrderPrice); return &v }(),
			ChangedBy: "system",
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}

	// Audit per take_profit_price
	if (oldOrder.TakeProfitPrice == nil && newOrder.TakeProfitPrice != nil) ||
		(oldOrder.TakeProfitPrice != nil && newOrder.TakeProfitPrice == nil) ||
		(oldOrder.TakeProfitPrice != nil && newOrder.TakeProfitPrice != nil && *oldOrder.TakeProfitPrice != *newOrder.TakeProfitPrice) {
		audit := &models.OrderAudit{
			OrderID:   newOrder.OrderID,
			FieldName: "take_profit_price",
			OldValue: func() *string {
				if oldOrder.TakeProfitPrice == nil {
					return nil
				}
				v := fmt.Sprintf("%.8f", *oldOrder.TakeProfitPrice)
				return &v
			}(),
			NewValue: func() *string {
				if newOrder.TakeProfitPrice == nil {
					return nil
				}
				v := fmt.Sprintf("%.8f", *newOrder.TakeProfitPrice)
				return &v
			}(),
			ChangedBy: "system",
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}

	// Audit per stop_loss_price
	if (oldOrder.StopLossPrice == nil && newOrder.StopLossPrice != nil) ||
		(oldOrder.StopLossPrice != nil && newOrder.StopLossPrice == nil) ||
		(oldOrder.StopLossPrice != nil && newOrder.StopLossPrice != nil && *oldOrder.StopLossPrice != *newOrder.StopLossPrice) {
		audit := &models.OrderAudit{
			OrderID:   newOrder.OrderID,
			FieldName: "stop_loss_price",
			OldValue: func() *string {
				if oldOrder.StopLossPrice == nil {
					return nil
				}
				v := fmt.Sprintf("%.8f", *oldOrder.StopLossPrice)
				return &v
			}(),
			NewValue: func() *string {
				if newOrder.StopLossPrice == nil {
					return nil
				}
				v := fmt.Sprintf("%.8f", *newOrder.StopLossPrice)
				return &v
			}(),
			ChangedBy: "system",
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}

	// Audit per order_status_id
	if oldOrder.OrderStatusID != newOrder.OrderStatusID {
		audit := &models.OrderAudit{
			OrderID:   newOrder.OrderID,
			FieldName: "order_status_id",
			OldValue:  func() *string { v := fmt.Sprintf("%d", oldOrder.OrderStatusID); return &v }(),
			NewValue:  func() *string { v := fmt.Sprintf("%d", newOrder.OrderStatusID); return &v }(),
			ChangedBy: "system",
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}

	// Audit per result
	if oldOrder.Result != newOrder.Result {
		audit := &models.OrderAudit{
			OrderID:   newOrder.OrderID,
			FieldName: "result",
			OldValue:  func() *string { v := string(oldOrder.Result); return &v }(),
			NewValue:  func() *string { v := string(newOrder.Result); return &v }(),
			ChangedBy: "system",
		}
		if err := tx.Create(audit).Error; err != nil {
			return err
		}
	}

	return nil
}
