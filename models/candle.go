package models

import "time"

// Market rappresenta il tipo di mercato (spot o derivati)
type Market string

const (
	SpotMarket        Market = "spot"
	DerivativesMarket Market = "derivatives"
)

// Timeframe rappresenta l'intervallo temporale di una candela
type Timeframe string

const (
	Timeframe1m  Timeframe = "1"
	Timeframe5m  Timeframe = "5"
	Timeframe15m Timeframe = "15"
	Timeframe30m Timeframe = "30"
	Timeframe1h  Timeframe = "60"
	Timeframe4h  Timeframe = "240"
	Timeframe1d  Timeframe = "D"
	Timeframe1w  Timeframe = "W"
	Timeframe1M  Timeframe = "M"
)

// Candle rappresenta una singola candela OHLCV
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// CandleResponse rappresenta la risposta paginata delle candele
type CandleResponse struct {
	Candles []Candle `json:"candles"`
	HasMore bool     `json:"has_more"`
}
