package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderParams rappresenta i parametri per la creazione di un ordine
type OrderParams struct {
	Symbol      string   `json:"symbol"`
	Side        string   `json:"side"`
	OrderType   string   `json:"orderType"`
	Price       float64  `json:"price"`
	Quantity    float64  `json:"qty"`
	TakeProfit  *float64 `json:"takeProfit,omitempty"`
	StopLoss    *float64 `json:"stopLoss,omitempty"`
	PositionIdx int      `json:"positionIdx"`           // 0: One-Way Mode, 1: Buy side of Hedge Mode, 2: Sell side of Hedge Mode
	OrderLinkId string   `json:"orderLinkId,omitempty"` // ID personalizzato per l'ordine
	TimeInForce string   `json:"timeInForce"`           // GTC, IOC, FOK, etc.
}

// OrderSideType rappresenta il lato dell'ordine (Buy/Sell)
type OrderSideType string

const (
	OrderSideTypeBuy  OrderSideType = "Buy"
	OrderSideTypeSell OrderSideType = "Sell"
)

// OrderResult rappresenta il risultato finale dell'ordine
type OrderResult string

const (
	OrderResultProfit  OrderResult = "Profit"
	OrderResultLoss    OrderResult = "Loss"
	OrderResultPending OrderResult = "Pending"
	OrderResultDone    OrderResult = "Done"
)

// Order rappresenta un ordine di trading nel sistema
type Order struct {
	// Chiave primaria auto-incrementale per performance
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// ID ordine da Bybit (univoco)
	OrderID string `gorm:"type:varchar(50);not null;uniqueIndex:idx_order_id" json:"order_id"`

	// Informazioni trading
	Symbol string        `gorm:"type:varchar(20);not null;index:idx_symbol;comment:Simbolo del trading pair (es. BTCUSDT)" json:"symbol"`
	Side   OrderSideType `gorm:"type:varchar(4);not null;index:idx_side;comment:Buy = LONG, Sell = SHORT" json:"side"`

	// Prezzi e quantità
	OrderPrice      float64  `gorm:"type:REAL;not null;comment:Prezzo dell'ordine" json:"order_price"`
	Quantity        float64  `gorm:"type:REAL;not null;comment:Quantità dell'ordine" json:"quantity"`
	TakeProfitPrice *float64 `gorm:"type:REAL;comment:Prezzo take profit" json:"take_profit_price"`
	StopLossPrice   *float64 `gorm:"type:REAL;comment:Prezzo stop loss" json:"stop_loss_price"`

	// Stato e risultato
	OrderStatusID uint               `gorm:"not null;index:idx_order_status_id" json:"order_status_id"`
	OrderStatus   *OrderStatusEntity `gorm:"foreignKey:OrderStatusID;references:ID;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE" json:"order_status,omitempty"`
	Result        OrderResult        `gorm:"type:varchar(10);default:'Pending';index:idx_result;comment:Risultato finale dell'ordine" json:"result"`

	// Metadati aggiuntivi per analisi
	PnL           float64 `gorm:"type:REAL;default:0.00000000;index:idx_pnl;comment:Profit and Loss calcolato" json:"pnl"`
	PnLPercentage float64 `gorm:"type:REAL;default:0.0000;index:idx_pnl_percentage;comment:PnL in percentuale" json:"pnl_percentage"`

	// Timestamps
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index:idx_created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index:idx_updated_at" json:"updated_at"`
}

// TableName specifica il nome della tabella per GORM
func (Order) TableName() string {
	return "orders"
}

// BeforeCreate hook per validazioni prima della creazione
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	// Validazione base
	if o.OrderID == "" || o.Symbol == "" || o.OrderPrice <= 0 || o.Quantity <= 0 {
		return gorm.ErrInvalidData
	}

	// Validazione side
	if o.Side != OrderSideTypeBuy && o.Side != OrderSideTypeSell {
		return gorm.ErrInvalidData
	}

	// Validazione result
	if o.Result != OrderResultProfit && o.Result != OrderResultLoss && o.Result != OrderResultPending {
		o.Result = OrderResultPending
	}

	// Validazione prezzi take profit e stop loss
	if err := o.validatePrices(); err != nil {
		return err
	}

	return nil
}

// BeforeUpdate hook per validazioni prima dell'aggiornamento
func (o *Order) BeforeUpdate(tx *gorm.DB) error {
	// Validazione prezzi take profit e stop loss
	if err := o.validatePrices(); err != nil {
		return err
	}

	return nil
}

// validatePrices valida i prezzi take profit e stop loss secondo la business logic
func (o *Order) validatePrices() error {
	// Validazione take profit
	if o.TakeProfitPrice != nil {
		if *o.TakeProfitPrice <= 0 {
			return gorm.ErrInvalidData
		}

		if o.Side == OrderSideTypeBuy && *o.TakeProfitPrice <= o.OrderPrice {
			return gorm.ErrInvalidData
		}

		if o.Side == OrderSideTypeSell && *o.TakeProfitPrice >= o.OrderPrice {
			return gorm.ErrInvalidData
		}
	}

	// Validazione stop loss
	if o.StopLossPrice != nil {
		if *o.StopLossPrice <= 0 {
			return gorm.ErrInvalidData
		}

		if o.Side == OrderSideTypeBuy && *o.StopLossPrice >= o.OrderPrice {
			return gorm.ErrInvalidData
		}

		if o.Side == OrderSideTypeSell && *o.StopLossPrice <= o.OrderPrice {
			return gorm.ErrInvalidData
		}
	}

	return nil
}

// IsActive verifica se l'ordine è attivo (non completato o cancellato)
func (o *Order) IsActive() bool {
	if o.OrderStatus == nil {
		return false
	}
	return o.Result == OrderResultPending &&
		(o.OrderStatus.StatusName == "New" ||
			o.OrderStatus.StatusName == "PartiallyFilled" ||
			o.OrderStatus.StatusName == "Untriggered" ||
			o.OrderStatus.StatusName == "Triggered")
}

// IsCompleted verifica se l'ordine è completato
func (o *Order) IsCompleted() bool {
	return o.Result != OrderResultPending
}

// IsProfitable verifica se l'ordine è profittevole
func (o *Order) IsProfitable() bool {
	return o.Result == OrderResultProfit
}

// IsLosing verifica se l'ordine è in perdita
func (o *Order) IsLosing() bool {
	return o.Result == OrderResultLoss
}

// CalculatePnL calcola il PnL basato sui prezzi (da implementare con logica specifica)
func (o *Order) CalculatePnL(currentPrice float64) {
	if o.Side == OrderSideTypeBuy {
		o.PnL = (currentPrice - o.OrderPrice) * o.Quantity
	} else {
		o.PnL = (o.OrderPrice - currentPrice) * o.Quantity
	}

	if o.OrderPrice != 0 {
		o.PnLPercentage = (o.PnL / (o.OrderPrice * o.Quantity)) * 100
	}
}

// String restituisce una rappresentazione stringa dell'ordine
func (o *Order) String() string {
	return o.OrderID
}
