package taprocess

import "cross-exchange-arbitrage/models"

// TAProcessor definisce l'interfaccia per il calcolo degli indicatori tecnici
type TAProcessor interface {
	// ProcessIndicators calcola tutti gli indicatori tecnici per i prezzi di chiusura forniti
	// Restituisce una slice di TACandlestick con gli indicatori calcolati
	ProcessIndicators(closingPrices []float64) ([]*models.TACandlestick, error)
}
