package exchange

import (
	"context"
	"cross-exchange-arbitrage/models"
)

// Exchange definisce l'interfaccia comune per tutti gli exchange
type Exchange interface {
	// GetRealTimePrice restituisce il prezzo in tempo reale con liquidità per una coppia di trading
	GetRealTimePrice(ctx context.Context, symbol string) (*models.RealTimePriceData, error)
}
