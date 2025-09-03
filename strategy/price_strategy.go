package strategy

// PriceStrategy definisce l'interfaccia per le strategie di calcolo del prezzo
type PriceStrategy interface {
	// GetPrice calcola il prezzo basato su due valori forniti
	GetPrice(value1, value2 float64) float64
}

// OrderPriceStrategy implementa l'interfaccia PriceStrategy
// Calcola la media aritmetica di due numeri
type OrderPriceStrategy struct{}

// NewOrderPriceStrategy crea una nuova istanza di OrderPriceStrategy
func NewOrderPriceStrategy() *OrderPriceStrategy {
	return &OrderPriceStrategy{}
}

// GetPrice implementa l'interfaccia PriceStrategy
// Ritorna la media aritmetica di due valori: (value1 + value2) / 2
// Esempio: GetPrice(5, 1) = (5 + 1) / 2 = 3
func (ops *OrderPriceStrategy) GetPrice(value1, value2 float64) float64 {
	return (value1 + value2) / 2.0
}
