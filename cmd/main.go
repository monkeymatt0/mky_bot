package main

import (
	"cross-exchange-arbitrage/worker"
)

// Esempio di utilizzo del nuovo sistema worker con cron
func main() {
	// Avvia il sistema worker completo
	// Questo sostituisce il vecchio sistema con ticker
	worker.StartWorkerSystem()
}
