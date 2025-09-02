package strategy

import "cross-exchange-arbitrage/models"

// Strategy definisce l'interfaccia per le strategie di trading
type Strategy interface {
	// OrderPrice calcola il prezzo di entrata basato sulla strategia fornita
	OrderPrice(strategyType string) (float64, error)

	// PlaceShortOrder piazza un ordine short con i parametri specificati
	CreateShortOrder(symbol string, price float64, quantity float64, takeProfit *float64, stopLoss *float64) error

	// PlaceLongOrder piazza un ordine long con i parametri specificati
	CreateLongOrder(symbol string, price float64, quantity float64, takeProfit *float64, stopLoss *float64) error

	// PlaceOrder piazza un ordine con i parametri specificati
	placeOrder(orderParams models.OrderParams) error // -> Questo è privato all'interno della struttura Strategy, volutamente
}
