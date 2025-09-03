package orderprocessor

import (
	"context"
	"cross-exchange-arbitrage/models"
)

// OrderProcessor definisce l'interfaccia per il piazzamento di ordini sui mercati derivati
type OrderProcessor interface {
	// PlaceLongOrder piazza un ordine long condizionale
	// L'ordine viene eseguito quando il prezzo raggiunge il prezzo specificato (trigger al rialzo)
	PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error)

	// PlaceShortOrder piazza un ordine short condizionale
	// L'ordine viene eseguito quando il prezzo raggiunge il prezzo specificato (trigger al ribasso)
	PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error)
}
