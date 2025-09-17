package models

import "time"

// OrderSide rappresenta il lato dell'ordine (Buy/Sell)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "Buy"
	OrderSideSell OrderSide = "Sell"
)

// OrderType rappresenta il tipo di ordine
type OrderType string

const (
	OrderTypeMarket     OrderType = "Market"
	OrderTypeLimit      OrderType = "Limit"
	OrderTypeStop       OrderType = "Stop"
	OrderTypeStopLimit  OrderType = "StopLimit"
	OrderTypeOpenLong   OrderType = "OpenLong"
	OrderTypeOpenShort  OrderType = "OpenShort"
	OrderTypeCloseLong  OrderType = "CloseLong"
	OrderTypeCloseShort OrderType = "CloseShort"
	OrderTypeStopLoss   OrderType = "StopLoss"
	OrderTypeTakeProfit OrderType = "TakeProfit"
)

// OrderStatus rappresenta lo stato dell'ordine
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "New"
	OrderStatusPartiallyFilled OrderStatus = "PartiallyFilled"
	OrderStatusFilled          OrderStatus = "Filled"
	OrderStatusCancelled       OrderStatus = "Cancelled"
	OrderStatusRejected        OrderStatus = "Rejected"
	OrderStatusUntriggered     OrderStatus = "Untriggered"
	OrderStatusTriggered       OrderStatus = "Triggered"
)

// TimeInForce rappresenta la durata dell'ordine
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
)

// TriggerDirection rappresenta la direzione del trigger per ordini condizionali
type TriggerDirection string

const (
	TriggerDirectionRising  TriggerDirection = "1" // Trigger quando il prezzo sale (per Long)
	TriggerDirectionFalling TriggerDirection = "2" // Trigger quando il prezzo scende (per Short)
)

type TriggerType string

const (
	TriggerTypePrice TriggerType = "LastPrice"
	TriggerTypeIndex TriggerType = "IndexPrice"
	TriggerTypeMark  TriggerType = "MarkPrice"
)

// OrderRequest rappresenta una richiesta di ordine per Bybit
type OrderRequest struct {
	Category         string           `json:"category"`                   // "linear" per derivatives perpetual
	Symbol           string           `json:"symbol"`                     // Es. "BTCUSDT"
	Side             OrderSide        `json:"side"`                       // "Buy" o "Sell"
	OrderType        OrderType        `json:"orderType"`                  // "Market", "Limit", "Stop", ecc.
	Qty              string           `json:"qty"`                        // Quantità
	Price            string           `json:"price,omitempty"`            // Prezzo (per ordini limit)
	TriggerPrice     string           `json:"triggerPrice,omitempty"`     // Prezzo trigger (per ordini stop)
	TriggerDirection TriggerDirection `json:"triggerDirection,omitempty"` // Direzione trigger (1=rising, 2=falling)
	StopLoss         string           `json:"stopLoss,omitempty"`         // Stop Loss
	TakeProfit       string           `json:"takeProfit,omitempty"`       // Take Profit
	TimeInForce      TimeInForce      `json:"timeInForce,omitempty"`      // Durata ordine
	OrderLinkId      string           `json:"orderLinkId,omitempty"`      // ID cliente per tracking
	TriggerBy        TriggerType      `json:"triggerBy,omitempty"`        // Tipo trigger (LastPrice, IndexPrice, MarkPrice)
	ReduceOnly       bool             `json:"reduceOnly,omitempty"`       // Reduce Only
}

// OrderResponse rappresenta la risposta di un ordine piazzato
type OrderResponse struct {
	OrderID      string      `json:"orderId"`
	OrderLinkID  string      `json:"orderLinkId"`
	Symbol       string      `json:"symbol"`
	Side         OrderSide   `json:"side"`
	OrderType    OrderType   `json:"orderType"`
	Price        float64     `json:"price"`
	AveragePrice float64     `json:"avgPrice"`
	Quantity     float64     `json:"qty"`
	Status       OrderStatus `json:"orderStatus"`
	TriggerPrice float64     `json:"triggerPrice,omitempty"`
	StopLoss     float64     `json:"stopLoss,omitempty"`
	TakeProfit   float64     `json:"takeProfit,omitempty"`
	CreatedTime  time.Time   `json:"createdTime"`
	UpdatedTime  time.Time   `json:"updatedTime"`
	ErrorCode    string      `json:"retCode,omitempty"`
	ErrorMessage string      `json:"retMsg,omitempty"`
}

// IsSuccess verifica se l'ordine è stato piazzato con successo
func (or *OrderResponse) IsSuccess() bool {
	return or.ErrorCode == "" || or.ErrorCode == "0"
}

// IsActive verifica se l'ordine è ancora attivo
func (or *OrderResponse) IsActive() bool {
	return or.Status == OrderStatusNew ||
		or.Status == OrderStatusPartiallyFilled ||
		or.Status == OrderStatusUntriggered
}

// IsFilled verifica se l'ordine è stato completamente riempito
func (or *OrderResponse) IsFilled() bool {
	return or.Status == OrderStatusFilled
}
