package taprocess

import (
	"cross-exchange-arbitrage/models"
	"fmt"
	"time"

	"github.com/markcheno/go-talib"
)

// TalibProcessor implementa TAProcessor usando la libreria go-talib
type TalibProcessor struct {
	// Configurazioni per i periodi degli indicatori
	RSIPeriod    int
	EMA20Period  int
	EMA60Period  int
	EMA223Period int
}

// NewTalibProcessor crea una nuova istanza di TalibProcessor con i periodi standard
func NewTalibProcessor() *TalibProcessor {
	return &TalibProcessor{
		RSIPeriod:    14,
		EMA20Period:  20,
		EMA60Period:  60,
		EMA223Period: 223,
	}
}

// ProcessIndicators implementa l'interfaccia TAProcessor
func (tp *TalibProcessor) ProcessIndicators(closingPrices []float64) ([]*models.TACandlestick, error) {
	if len(closingPrices) == 0 {
		return nil, fmt.Errorf("closingPrices slice è vuota")
	}

	// Verifica che abbiamo abbastanza dati per calcolare tutti gli indicatori
	minRequiredData := tp.EMA223Period // EMA223 richiede il maggior numero di dati
	if len(closingPrices) < minRequiredData {
		return nil, fmt.Errorf("dati insufficienti: richiesti almeno %d prezzi, ricevuti %d", minRequiredData, len(closingPrices))
	}

	// Calcola gli indicatori usando go-talib
	ema223Values := talib.Ema(closingPrices, tp.EMA223Period)
	ema20Values := talib.Ema(closingPrices, tp.EMA20Period)
	ema60Values := talib.Ema(closingPrices, tp.EMA60Period)
	rsi14Values := talib.Rsi(closingPrices, tp.RSIPeriod)

	// Crea la slice di risultati
	results := make([]*models.TACandlestick, len(closingPrices))

	// Per ogni prezzo di chiusura, crea un TACandlestick con gli indicatori
	for i, closePrice := range closingPrices {
		// Crea un timestamp fittizio (sarà sostituito con dati reali quando necessario)
		timestamp := time.Now().Add(time.Duration(-len(closingPrices)+i+1) * time.Minute)

		// Crea la candela base con dati OHLCV fittizi (solo il Close è reale)
		taCandlestick := &models.TACandlestick{
			Timestamp: timestamp,
			Open:      closePrice, // Valori fittizi
			High:      closePrice,
			Low:       closePrice,
			Close:     closePrice,
			Volume:    0, // Valore fittizio
		}

		// Imposta gli indicatori se disponibili (go-talib restituisce NaN per valori non calcolabili)
		var ema223, ema20, ema60, rsi14 *float64

		if i < len(ema223Values) && !isNaN(ema223Values[i]) {
			val := ema223Values[i]
			ema223 = &val
		}

		if i < len(ema20Values) && !isNaN(ema20Values[i]) {
			val := ema20Values[i]
			ema20 = &val
		}

		if i < len(ema60Values) && !isNaN(ema60Values[i]) {
			val := ema60Values[i]
			ema60 = &val
		}

		if i < len(rsi14Values) && !isNaN(rsi14Values[i]) {
			val := rsi14Values[i]
			rsi14 = &val
		}

		taCandlestick.SetIndicators(ema223, ema20, ema60, rsi14)
		results[i] = taCandlestick
	}

	return results, nil
}

// ProcessCandlesWithIndicators prende candele esistenti e calcola gli indicatori
func (tp *TalibProcessor) ProcessCandlesWithIndicators(candles []models.Candle) ([]*models.TACandlestick, error) {
	if len(candles) == 0 {
		return nil, fmt.Errorf("candles slice è vuota")
	}

	// Estrai i prezzi di chiusura
	closingPrices := make([]float64, len(candles))
	for i, candle := range candles {
		closingPrices[i] = candle.Close
	}

	// Calcola gli indicatori
	ema223Values := talib.Ema(closingPrices, tp.EMA223Period)
	ema20Values := talib.Ema(closingPrices, tp.EMA20Period)
	ema60Values := talib.Ema(closingPrices, tp.EMA60Period)
	rsi14Values := talib.Rsi(closingPrices, tp.RSIPeriod)

	// Crea la slice di risultati mantenendo i dati OHLCV originali
	results := make([]*models.TACandlestick, len(candles))

	for i, candle := range candles {
		// Crea TACandlestick dai dati originali della candela
		taCandlestick := models.NewTACandlestickFromCandle(candle)

		// Imposta gli indicatori se disponibili
		var ema223, ema20, ema60, rsi14 *float64

		if i < len(ema223Values) && !isNaN(ema223Values[i]) {
			val := ema223Values[i]
			ema223 = &val
		}

		if i < len(ema20Values) && !isNaN(ema20Values[i]) {
			val := ema20Values[i]
			ema20 = &val
		}

		if i < len(ema60Values) && !isNaN(ema60Values[i]) {
			val := ema60Values[i]
			ema60 = &val
		}

		if i < len(rsi14Values) && !isNaN(rsi14Values[i]) {
			val := rsi14Values[i]
			rsi14 = &val
		}

		taCandlestick.SetIndicators(ema223, ema20, ema60, rsi14)
		results[i] = taCandlestick
	}

	return results, nil
}

// isNaN verifica se un float64 è NaN (Not a Number)
func isNaN(f float64) bool {
	return f != f
}
