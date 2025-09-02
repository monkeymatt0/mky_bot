package models

import "time"

// OrderBookLevel rappresenta un livello dell'order book
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// OrderBookData rappresenta i dati dell'order book con liquidità
type OrderBookData struct {
	Symbol    string           `json:"symbol"`
	BestBid   OrderBookLevel   `json:"best_bid"`
	BestAsk   OrderBookLevel   `json:"best_ask"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	Exchange  string           `json:"exchange"`
	Timestamp time.Time        `json:"timestamp"`
}

// RealTimePriceData rappresenta i dati di prezzo in tempo reale con liquidità
type RealTimePriceData struct {
	Symbol       string    `json:"symbol"`
	Price        float64   `json:"price"`         // Prezzo medio tra bid e ask
	BidPrice     float64   `json:"bid_price"`     // Miglior prezzo di acquisto
	AskPrice     float64   `json:"ask_price"`     // Miglior prezzo di vendita
	BidLiquidity float64   `json:"bid_liquidity"` // Liquidità al miglior bid
	AskLiquidity float64   `json:"ask_liquidity"` // Liquidità al miglior ask
	Exchange     string    `json:"exchange"`
	Timestamp    time.Time `json:"timestamp"`
}
