package repositories

import (
	"context"
	"cross-exchange-arbitrage/models"

	"gorm.io/gorm"
)

// orderRepository implementa OrderRepository
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository crea una nuova istanza di OrderRepository
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Create crea un nuovo ordine
func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// GetByID recupera un ordine per ID
func (r *orderRepository) GetByID(ctx context.Context, id uint) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Preload("OrderStatus").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderID recupera un ordine per OrderID (ID Bybit)
func (r *orderRepository) GetByOrderID(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Preload("OrderStatus").Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetAll recupera tutti gli ordini con paginazione
func (r *orderRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetBySymbol recupera ordini per simbolo
func (r *orderRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").Where("symbol = ?", symbol)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetByStatus recupera ordini per stato
func (r *orderRepository) GetByStatus(ctx context.Context, statusName string, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").Joins("JOIN order_statuses ON orders.order_status_id = order_statuses.id").
		Where("order_statuses.status_name = ?", statusName)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("orders.created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetByResult recupera ordini per risultato
func (r *orderRepository) GetByResult(ctx context.Context, result models.OrderResult, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").Where("result = ?", result)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetActiveOrders recupera ordini attivi
func (r *orderRepository) GetActiveOrders(ctx context.Context, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").
		Joins("JOIN order_statuses ON orders.order_status_id = order_statuses.id").
		Where("order_statuses.status_name IN ? AND orders.result = ?",
			[]string{"New", "PartiallyFilled", "Untriggered", "Triggered"},
			models.OrderResultPending)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("orders.created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetBySymbolAndStatus recupera ordini per simbolo e stato
func (r *orderRepository) GetBySymbolAndStatus(ctx context.Context, symbol, statusName string, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").
		Joins("JOIN order_statuses ON orders.order_status_id = order_statuses.id").
		Where("orders.symbol = ? AND order_statuses.status_name = ?", symbol, statusName)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("orders.created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetBySymbolAndResult recupera ordini per simbolo e risultato
func (r *orderRepository) GetBySymbolAndResult(ctx context.Context, symbol string, result models.OrderResult, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").
		Where("symbol = ? AND result = ?", symbol, result)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// GetByDateRange recupera ordini in un range di date
func (r *orderRepository) GetByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*models.Order, error) {
	var orders []*models.Order
	query := r.db.WithContext(ctx).Preload("OrderStatus").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Update aggiorna un ordine esistente
func (r *orderRepository) Update(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// UpdateStatus aggiorna solo lo stato di un ordine
func (r *orderRepository) UpdateStatus(ctx context.Context, orderID string, statusID uint) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).
		Where("order_id = ?", orderID).
		Update("order_status_id", statusID).Error
}

// UpdateResult aggiorna solo il risultato di un ordine
func (r *orderRepository) UpdateResult(ctx context.Context, orderID string, result models.OrderResult) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).
		Where("order_id = ?", orderID).
		Update("result", result).Error
}

// UpdatePnL aggiorna PnL e PnL percentage di un ordine
func (r *orderRepository) UpdatePnL(ctx context.Context, orderID string, pnl, pnlPercentage float64) error {
	return r.db.WithContext(ctx).Model(&models.Order{}).
		Where("order_id = ?", orderID).
		Updates(map[string]interface{}{
			"pnl":            pnl,
			"pnl_percentage": pnlPercentage,
		}).Error
}

// UpdatePrices aggiorna prezzi take profit e stop loss
func (r *orderRepository) UpdatePrices(ctx context.Context, orderID string, takeProfit, stopLoss *float64) error {
	updates := map[string]interface{}{}
	if takeProfit != nil {
		updates["take_profit_price"] = *takeProfit
	}
	if stopLoss != nil {
		updates["stop_loss_price"] = *stopLoss
	}

	return r.db.WithContext(ctx).Model(&models.Order{}).
		Where("order_id = ?", orderID).
		Updates(updates).Error
}

// Delete elimina un ordine (soft delete)
func (r *orderRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Order{}, id).Error
}

// Exists verifica se un ordine esiste
func (r *orderRepository) Exists(ctx context.Context, orderID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Order{}).Where("order_id = ?", orderID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count conta il numero totale di ordini
func (r *orderRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Order{}).Count(&count).Error
	return count, err
}

// CountBySymbol conta ordini per simbolo
func (r *orderRepository) CountBySymbol(ctx context.Context, symbol string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Order{}).Where("symbol = ?", symbol).Count(&count).Error
	return count, err
}

// CountByStatus conta ordini per stato
func (r *orderRepository) CountByStatus(ctx context.Context, statusName string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Order{}).
		Joins("JOIN order_statuses ON orders.order_status_id = order_statuses.id").
		Where("order_statuses.status_name = ?", statusName).Count(&count).Error
	return count, err
}

// CountByResult conta ordini per risultato
func (r *orderRepository) CountByResult(ctx context.Context, result models.OrderResult) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Order{}).Where("result = ?", result).Count(&count).Error
	return count, err
}

// GetTradingStats recupera statistiche di trading
func (r *orderRepository) GetTradingStats(ctx context.Context, symbol string) (*TradingStats, error) {
	var stats TradingStats

	query := r.db.WithContext(ctx).Model(&models.Order{})
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}

	err := query.Select(`
		symbol,
		side,
		COUNT(*) as total_orders,
		SUM(CASE WHEN result = ? THEN 1 ELSE 0 END) as profitable_orders,
		SUM(CASE WHEN result = ? THEN 1 ELSE 0 END) as losing_orders,
		SUM(CASE WHEN result = ? THEN 1 ELSE 0 END) as pending_orders,
		AVG(pnl) as avg_pnl,
		AVG(pnl_percentage) as avg_pnl_percentage,
		SUM(pnl) as total_pnl
	`, models.OrderResultProfit, models.OrderResultLoss, models.OrderResultPending).
		Group("symbol, side").
		First(&stats).Error

	if err != nil {
		return nil, err
	}

	// Calcola win rate
	if stats.TotalOrders > 0 {
		stats.WinRate = float64(stats.ProfitableOrders) / float64(stats.TotalOrders) * 100
	}

	return &stats, nil
}

// GetPnLStats recupera statistiche PnL
func (r *orderRepository) GetPnLStats(ctx context.Context, symbol string) (*PnLStats, error) {
	var stats PnLStats

	query := r.db.WithContext(ctx).Model(&models.Order{})
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}

	err := query.Select(`
		symbol,
		SUM(pnl) as total_pnl,
		AVG(pnl) as avg_pnl,
		MAX(pnl) as max_pnl,
		MIN(pnl) as min_pnl,
		SUM(pnl_percentage) as total_pnl_percentage,
		AVG(pnl_percentage) as avg_pnl_percentage,
		MAX(pnl_percentage) as max_pnl_percentage,
		MIN(pnl_percentage) as min_pnl_percentage
	`).Group("symbol").First(&stats).Error

	if err != nil {
		return nil, err
	}

	return &stats, nil
}
