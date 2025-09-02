package exchange

import (
	"context"
	"cross-exchange-arbitrage/models"
)

// Exchange definisce l'interfaccia comune per tutti gli exchange
type Exchange interface {
	// GetRealTimePrice restituisce il prezzo in tempo reale con liquidità per una coppia di trading
	GetRealTimePrice(ctx context.Context, symbol string) (*models.RealTimePriceData, error)

	// FetchLastCandles recupera le candele storiche per un determinato simbolo
	// Se market non è specificato, usa il mercato derivatives perpetual di default
	// La funzione gestisce automaticamente la paginazione e il rate limiting
	FetchLastCandles(ctx context.Context, symbol string, market models.Market, timeframe models.Timeframe, limit int) (*models.CandleResponse, error)
}
