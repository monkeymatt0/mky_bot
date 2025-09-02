package models

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
