package models

import "time"

// Execution rappresenta un singolo trade/esecuzione
type Execution struct {
	Symbol      string    `json:"symbol" gorm:"column:symbol"`
	Side        string    `json:"side" gorm:"column:side"`
	OrderID     string    `json:"orderId" gorm:"column:order_id"`
	ExecID      string    `json:"execId" gorm:"column:exec_id"`
	Price       float64   `json:"price" gorm:"column:price"`
	Qty         float64   `json:"qty" gorm:"column:qty"`
	ExecType    string    `json:"execType" gorm:"column:exec_type"`
	ExecTime    time.Time `json:"execTime" gorm:"column:exec_time"`
	IsMaker     bool      `json:"isMaker" gorm:"column:is_maker"`
	Fee         float64   `json:"fee" gorm:"column:fee"`
	FeeCurrency string    `json:"feeCurrency" gorm:"column:fee_currency"`
	TradeTime   time.Time `json:"tradeTime" gorm:"column:trade_time"`
	Exchange    string    `json:"exchange" gorm:"column:exchange"`
}

// TableName specifica il nome della tabella per GORM
func (Execution) TableName() string {
	return "executions"
}

// ExecutionResponse rappresenta la risposta per una lista di esecuzioni
type ExecutionResponse struct {
	Executions []Execution `json:"executions"`
	HasMore    bool        `json:"hasMore"`
	Total      int         `json:"total"`
}
