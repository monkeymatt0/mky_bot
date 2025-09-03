package models

import "time"

// TACandlestick rappresenta una candela con i dati OHLCV e gli indicatori tecnici
type TACandlestick struct {
	// Dati OHLCV della candela
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`

	// Indicatori tecnici
	EMA223 *float64 `json:"ema223,omitempty"` // Puntatore per gestire valori non calcolabili (es. dati insufficienti)
	EMA20  *float64 `json:"ema20,omitempty"`
	EMA60  *float64 `json:"ema60,omitempty"`
	RSI14  *float64 `json:"rsi14,omitempty"`
}

// NewTACandlestickFromCandle crea un TACandlestick da una Candle esistente
func NewTACandlestickFromCandle(candle Candle) *TACandlestick {
	return &TACandlestick{
		Timestamp: candle.Timestamp,
		Open:      candle.Open,
		High:      candle.High,
		Low:       candle.Low,
		Close:     candle.Close,
		Volume:    candle.Volume,
		EMA223:    nil, // Gli indicatori verranno calcolati successivamente
		EMA20:     nil,
		EMA60:     nil,
		RSI14:     nil,
	}
}

// HasAllIndicators verifica se tutti gli indicatori tecnici sono stati calcolati
func (tc *TACandlestick) HasAllIndicators() bool {
	return tc.EMA223 != nil && tc.EMA20 != nil && tc.EMA60 != nil && tc.RSI14 != nil
}

// GetEMA223 restituisce il valore EMA223 o 0 se non calcolato
func (tc *TACandlestick) GetEMA223() float64 {
	if tc.EMA223 == nil {
		return 0
	}
	return *tc.EMA223
}

// GetEMA20 restituisce il valore EMA20 o 0 se non calcolato
func (tc *TACandlestick) GetEMA20() float64 {
	if tc.EMA20 == nil {
		return 0
	}
	return *tc.EMA20
}

// GetEMA60 restituisce il valore EMA60 o 0 se non calcolato
func (tc *TACandlestick) GetEMA60() float64 {
	if tc.EMA60 == nil {
		return 0
	}
	return *tc.EMA60
}

// GetRSI14 restituisce il valore RSI14 o 0 se non calcolato
func (tc *TACandlestick) GetRSI14() float64 {
	if tc.RSI14 == nil {
		return 0
	}
	return *tc.RSI14
}

// SetIndicators imposta tutti gli indicatori tecnici
func (tc *TACandlestick) SetIndicators(ema223, ema20, ema60, rsi14 *float64) {
	tc.EMA223 = ema223
	tc.EMA20 = ema20
	tc.EMA60 = ema60
	tc.RSI14 = rsi14
}
