package orderprocessor

import (
	"context"
	"cross-exchange-arbitrage/models"
)

// UpdateOrderParams rappresenta i parametri per aggiornare un ordine
type UpdateOrderParams struct {
	Symbol      string   `json:"symbol"`               // Simbolo trading (es. "BTCUSDT")
	StopLoss    *float64 `json:"stopLoss,omitempty"`   // Nuovo prezzo stop loss (opzionale)
	TakeProfit  *float64 `json:"takeProfit,omitempty"` // Nuovo prezzo take profit (opzionale)
	PositionIdx int      `json:"positionIdx"`          // 0=One-Way Mode, 1=Long hedge, 2=Short hedge
}

// OrderProcessor definisce l'interfaccia per il piazzamento di ordini sui mercati derivati
type OrderProcessor interface {
	// PlaceLongOrder piazza un ordine long condizionale
	// L'ordine viene eseguito quando il prezzo raggiunge il prezzo specificato (trigger al rialzo)
	PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error)

	// PlaceShortOrder piazza un ordine short condizionale
	// L'ordine viene eseguito quando il prezzo raggiunge il prezzo specificato (trigger al ribasso)
	PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error)

	// DeleteOrder cancella un ordine esistente usando l'orderID o orderLinkID
	// Accetta sia l'ID dell'ordine di Bybit (UUID) che l'ID cliente personalizzato
	DeleteOrder(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error)

	// UpdateOrder aggiorna stop loss e/o take profit di una posizione esistente
	// Accetta parametri flessibili - può aggiornare solo SL, solo TP, o entrambi
	UpdateOrder(ctx context.Context, params UpdateOrderParams) (*models.OrderResponse, error)

	// GetOrderStatus recupera lo stato corrente di un ordine
	// Accetta sia orderID (UUID di Bybit) che orderLinkID (ID cliente personalizzato)
	GetOrderStatus(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error)

	// GetPositions recupera le posizioni attive per un simbolo specifico
	// Se symbol è vuoto, restituisce tutte le posizioni attive
	GetPositions(ctx context.Context, symbol string) ([]models.Position, error)

	// GetWalletBalance recupera il saldo del wallet per un account specifico
	// Se coin è vuoto, restituisce tutti i saldi; altrimenti filtra per la criptovaluta specificata
	GetWalletBalance(ctx context.Context, accountType, coin string) (*models.WalletBalanceResponse, error)

	// GetUSDTBalance recupera il saldo USDT dal wallet (metodo di convenienza)
	GetUSDTBalance(ctx context.Context) (float64, error)

	// GetCoinBalance recupera il saldo per una specifica criptovaluta (metodo di convenienza)
	GetCoinBalance(ctx context.Context, coin string) (float64, error)
}
