package models

import "time"

// Price rappresenta il prezzo di un asset su un exchange
type Price struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Exchange  string    `json:"exchange"`
	Timestamp time.Time `json:"timestamp"`
}
