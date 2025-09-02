package main

import (
	"context"
	"fmt"
	"log"

	"cross-exchange-arbitrage/config"
	"cross-exchange-arbitrage/exchange"
	"cross-exchange-arbitrage/models"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Carica configurazioni
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Errore nel caricamento della configurazione:", err)
	}

	fmt.Printf("Configurazione caricata con successo!\n")
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("Bybit API Key configurata: %t\n", cfg.Bybit.APIKey != "")

	// Inizializza Bybit Exchange
	bybitExchange := exchange.NewBybitExchange()
	defer bybitExchange.Close()

	fmt.Println("Inizializzazione Exchange Bybit...")

	// Test con BTCUSDT perpetuals
	symbol := "BTCUSDT"

	// Esempio di fetch delle candele storiche
	fmt.Printf("\nRecupero candele storiche per %s...\n", symbol)
	if err := fetchHistoricalData(ctx, bybitExchange, symbol); err != nil {
		log.Printf("Errore nel recupero delle candele storiche: %v", err)
	}

	fmt.Printf("\nAvvio monitoraggio in tempo reale per %s su Bybit Perpetuals...\n", symbol)
}

// fetchHistoricalData recupera e visualizza le candele storiche per un simbolo
func fetchHistoricalData(ctx context.Context, ex exchange.Exchange, symbol string) error {
	// Recupera le ultime 100 candele da 1 ora
	fmt.Printf("Recupero ultime 100 candele da 1 ora per %s...\n", symbol)
	candleResp, err := ex.FetchLastCandles(ctx, symbol, models.DerivativesMarket, models.Timeframe1h, 100)
	if err != nil {
		return fmt.Errorf("errore nel recupero candele da 1 ora: %w", err)
	}

	fmt.Printf("\nRecuperate %d candele da 1 ora:\n", len(candleResp.Candles))
	for i, candle := range candleResp.Candles {
		if i < 5 { // Mostra solo le prime 5 candele per brevità
			fmt.Printf("[%s] Open: %.2f, High: %.2f, Low: %.2f, Close: %.2f, Volume: %.2f\n",
				candle.Timestamp.Format("2006-01-02 15:04:05"),
				candle.Open,
				candle.High,
				candle.Low,
				candle.Close,
				candle.Volume)
		}
	}
	if len(candleResp.Candles) > 5 {
		fmt.Printf("... e altre %d candele\n", len(candleResp.Candles)-5)
	}

	// Recupera le ultime 1000 candele da 1 minuto per dimostrare la paginazione
	fmt.Printf("\nRecupero ultime 1000 candele da 1 minuto per %s...\n", symbol)
	candleResp, err = ex.FetchLastCandles(ctx, symbol, models.DerivativesMarket, models.Timeframe1m, 1000)
	if err != nil {
		return fmt.Errorf("errore nel recupero candele da 1 minuto: %w", err)
	}

	fmt.Printf("\nRecuperate %d candele da 1 minuto\n", len(candleResp.Candles))
	fmt.Printf("Prima candela: %s\n", candleResp.Candles[0].Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Ultima candela: %s\n", candleResp.Candles[len(candleResp.Candles)-1].Timestamp.Format("2006-01-02 15:04:05"))

	// Calcola alcune statistiche di base
	var totalVolume float64
	var highestPrice float64
	var lowestPrice = candleResp.Candles[0].Low // inizializza con il primo prezzo low

	for _, candle := range candleResp.Candles {
		totalVolume += candle.Volume
		if candle.High > highestPrice {
			highestPrice = candle.High
		}
		if candle.Low < lowestPrice {
			lowestPrice = candle.Low
		}
	}

	fmt.Printf("\nStatistiche per le candele da 1 minuto:\n")
	fmt.Printf("Volume totale: %.2f\n", totalVolume)
	fmt.Printf("Prezzo più alto: %.2f\n", highestPrice)
	fmt.Printf("Prezzo più basso: %.2f\n", lowestPrice)
	fmt.Printf("Range di prezzo: %.2f\n", highestPrice-lowestPrice)

	return nil
}
