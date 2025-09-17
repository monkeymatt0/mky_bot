package repositories

import (
	"context"
	"cross-exchange-arbitrage/models"

	"gorm.io/gorm"
)

// OrderStatusRepository definisce l'interfaccia per le operazioni CRUD sugli stati degli ordini
type OrderStatusRepository interface {
	// Create crea un nuovo stato ordine
	Create(ctx context.Context, orderStatus *models.OrderStatusEntity) error

	// GetByID recupera uno stato ordine per ID
	GetByID(ctx context.Context, id uint) (*models.OrderStatusEntity, error)

	// GetByStatusName recupera uno stato ordine per nome
	GetByStatusName(ctx context.Context, statusName string) (*models.OrderStatusEntity, error)

	// GetAll recupera tutti gli stati ordine
	GetAll(ctx context.Context) ([]*models.OrderStatusEntity, error)

	// GetActive recupera solo gli stati attivi
	GetActive(ctx context.Context) ([]*models.OrderStatusEntity, error)

	// Update aggiorna uno stato ordine esistente
	Update(ctx context.Context, orderStatus *models.OrderStatusEntity) error

	// Delete elimina uno stato ordine (soft delete)
	Delete(ctx context.Context, id uint) error

	// Exists verifica se uno stato ordine esiste
	Exists(ctx context.Context, id uint) (bool, error)
}

// OrderRepository definisce l'interfaccia per le operazioni CRUD sugli ordini
type OrderRepository interface {
	// Create crea un nuovo ordine
	Create(ctx context.Context, order *models.Order) error

	// GetByID recupera un ordine per ID
	GetByID(ctx context.Context, id uint) (*models.Order, error)

	// GetByOrderID recupera un ordine per OrderID (ID Bybit)
	GetByOrderID(ctx context.Context, orderID string) (*models.Order, error)

	// GetAll recupera tutti gli ordini con paginazione
	GetAll(ctx context.Context, limit, offset int) ([]*models.Order, error)

	// GetBySymbol recupera ordini per simbolo
	GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*models.Order, error)

	// GetByStatus recupera ordini per stato
	GetByStatus(ctx context.Context, statusName string, limit, offset int) ([]*models.Order, error)

	// GetByResult recupera ordini per risultato
	GetByResult(ctx context.Context, result models.OrderResult, limit, offset int) ([]*models.Order, error)

	// GetActiveOrders recupera ordini attivi
	GetActiveOrders(ctx context.Context, limit, offset int) ([]*models.Order, error)

	// GetBySymbolAndStatus recupera ordini per simbolo e stato
	GetBySymbolAndStatus(ctx context.Context, symbol, statusName string, limit, offset int) ([]*models.Order, error)

	// GetBySymbolAndResult recupera ordini per simbolo e risultato
	GetBySymbolAndResult(ctx context.Context, symbol string, result models.OrderResult, limit, offset int) ([]*models.Order, error)

	// GetByDateRange recupera ordini in un range di date
	GetByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*models.Order, error)

	// Update aggiorna un ordine esistente
	Update(ctx context.Context, order *models.Order) error

	// UpdateStatus aggiorna solo lo stato di un ordine
	UpdateStatus(ctx context.Context, orderID string, statusID uint) error

	// UpdateResult aggiorna solo il risultato di un ordine
	UpdateResult(ctx context.Context, orderID string, result models.OrderResult) error

	// UpdatePnL aggiorna PnL e PnL percentage di un ordine
	UpdatePnL(ctx context.Context, orderID string, pnl, pnlPercentage float64) error

	// UpdatePrices aggiorna prezzi take profit e stop loss
	UpdatePrices(ctx context.Context, orderID string, takeProfit, stopLoss *float64) error

	// Delete elimina un ordine (soft delete)
	Delete(ctx context.Context, id uint) error

	// Exists verifica se un ordine esiste
	Exists(ctx context.Context, orderID string) (bool, error)

	// Count conta il numero totale di ordini
	Count(ctx context.Context) (int64, error)

	// CountBySymbol conta ordini per simbolo
	CountBySymbol(ctx context.Context, symbol string) (int64, error)

	// CountByStatus conta ordini per stato
	CountByStatus(ctx context.Context, statusName string) (int64, error)

	// CountByResult conta ordini per risultato
	CountByResult(ctx context.Context, result models.OrderResult) (int64, error)

	// GetTradingStats recupera statistiche di trading
	GetTradingStats(ctx context.Context, symbol string) (*TradingStats, error)

	// GetPnLStats recupera statistiche PnL
	GetPnLStats(ctx context.Context, symbol string) (*PnLStats, error)
}

// OrderAuditRepository definisce l'interfaccia per le operazioni CRUD sull'audit trail
type OrderAuditRepository interface {
	// Create crea un nuovo record di audit
	Create(ctx context.Context, audit *models.OrderAudit) error

	// GetByOrderID recupera tutti i record di audit per un ordine
	GetByOrderID(ctx context.Context, orderID string, limit, offset int) ([]*models.OrderAudit, error)

	// GetByFieldName recupera record di audit per campo
	GetByFieldName(ctx context.Context, fieldName string, limit, offset int) ([]*models.OrderAudit, error)

	// GetByDateRange recupera record di audit in un range di date
	GetByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*models.OrderAudit, error)

	// GetByOrderIDAndField recupera record di audit per ordine e campo specifico
	GetByOrderIDAndField(ctx context.Context, orderID, fieldName string) ([]*models.OrderAudit, error)

	// CountByOrderID conta i record di audit per un ordine
	CountByOrderID(ctx context.Context, orderID string) (int64, error)

	// GetLatestByOrderID recupera l'ultimo record di audit per un ordine
	GetLatestByOrderID(ctx context.Context, orderID string) (*models.OrderAudit, error)

	// DeleteOldRecords elimina record di audit più vecchi di una data
	DeleteOldRecords(ctx context.Context, beforeDate string) error
}

// TradingStats rappresenta le statistiche di trading
type TradingStats struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	TotalOrders      int64   `json:"total_orders"`
	ProfitableOrders int64   `json:"profitable_orders"`
	LosingOrders     int64   `json:"losing_orders"`
	PendingOrders    int64   `json:"pending_orders"`
	AvgPnL           float64 `json:"avg_pnl"`
	AvgPnLPercentage float64 `json:"avg_pnl_percentage"`
	TotalPnL         float64 `json:"total_pnl"`
	WinRate          float64 `json:"win_rate"`
}

// PnLStats rappresenta le statistiche PnL
type PnLStats struct {
	Symbol             string  `json:"symbol"`
	TotalPnL           float64 `json:"total_pnl"`
	AvgPnL             float64 `json:"avg_pnl"`
	MaxPnL             float64 `json:"max_pnl"`
	MinPnL             float64 `json:"min_pnl"`
	TotalPnLPercentage float64 `json:"total_pnl_percentage"`
	AvgPnLPercentage   float64 `json:"avg_pnl_percentage"`
	MaxPnLPercentage   float64 `json:"max_pnl_percentage"`
	MinPnLPercentage   float64 `json:"min_pnl_percentage"`
}

// RepositoryManager gestisce tutti i repository
type RepositoryManager interface {
	// OrderStatus restituisce il repository per gli stati ordine
	OrderStatus() OrderStatusRepository

	// Order restituisce il repository per gli ordini
	Order() OrderRepository

	// OrderAudit restituisce il repository per l'audit trail
	OrderAudit() OrderAuditRepository

	// BeginTransaction inizia una transazione
	BeginTransaction(ctx context.Context) (*gorm.DB, error)

	// CommitTransaction committa una transazione
	CommitTransaction(tx *gorm.DB) error

	// RollbackTransaction fa rollback di una transazione
	RollbackTransaction(tx *gorm.DB) error
}
