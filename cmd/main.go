package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cross-exchange-arbitrage/config"
	"cross-exchange-arbitrage/exchange"
	"cross-exchange-arbitrage/models"
	"cross-exchange-arbitrage/taprocess"
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

	// Test degli indicatori tecnici
	fmt.Printf("\n" + strings.Repeat("=", 50))
	fmt.Printf("\nTEST INDICATORI TECNICI\n")
	fmt.Printf(strings.Repeat("=", 50) + "\n")
	if err := testTechnicalIndicators(ctx, bybitExchange, symbol); err != nil {
		log.Printf("Errore nel test degli indicatori tecnici: %v", err)
	}

	fmt.Printf("\nAvvio monitoraggio in tempo reale per %s su Bybit Perpetuals...\n", symbol)
}

// fetchHistoricalData recupera e visualizza le candele storiche per un simbolo
func fetchHistoricalData(ctx context.Context, ex exchange.Exchange, symbol string) error {
	// Recupera le ultime 100 candele da 1 ora
	fmt.Printf("Recupero ultime 100 candele da 1 ora per %s...\n", symbol)
	candleResp, err := ex.FetchLastCandles(ctx, symbol, models.DerivativesMarket, models.Timeframe1h, 2000)
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
	candleResp, err = ex.FetchLastCandles(ctx, symbol, models.DerivativesMarket, models.Timeframe1m, 3000)
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

// testTechnicalIndicators testa il calcolo degli indicatori tecnici
func testTechnicalIndicators(ctx context.Context, ex exchange.Exchange, symbol string) error {
	// 1) Creo il bybit exchange (già fatto nel main)
	fmt.Printf("1) Exchange Bybit già inizializzato\n")

	// 2) Fetch dei dati dall'exchange
	fmt.Printf("2) Recupero candele per calcolare indicatori tecnici...\n")

	// Recuperiamo abbastanza candele per calcolare EMA223 (serve almeno 223 candele)
	candleResp, err := ex.FetchLastCandles(ctx, symbol, models.DerivativesMarket, models.Timeframe1h, 500)
	if err != nil {
		return fmt.Errorf("errore nel recupero candele per indicatori: %w", err)
	}

	fmt.Printf("   - Recuperate %d candele da 1 ora per %s\n", len(candleResp.Candles), symbol)

	// 3) Processing degli indicatori tecnici
	fmt.Printf("3) Calcolo indicatori tecnici (EMA20, EMA60, EMA223, RSI14)...\n")

	// Inizializza il processor degli indicatori tecnici
	processor := taprocess.NewTalibProcessor()

	// Calcola gli indicatori dalle candele
	taCandlesticks, err := processor.ProcessCandlesWithIndicators(candleResp.Candles)
	if err != nil {
		return fmt.Errorf("errore nel calcolo degli indicatori: %w", err)
	}

	fmt.Printf("   - Indicatori calcolati per %d candele\n", len(taCandlesticks))

	// Stampa le ultime 5 candele con gli indicatori tecnici
	fmt.Printf("\n4) ULTIME 5 CANDELE CON INDICATORI TECNICI:\n")
	fmt.Printf(strings.Repeat("-", 80) + "\n")

	// Prende le ultime 5 candele
	startIdx := len(taCandlesticks) - 5
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(taCandlesticks); i++ {
		tc := taCandlesticks[i]

		fmt.Printf("\nCandela #%d [%s]:\n", i+1, tc.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  OHLCV: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f\n",
			tc.Open, tc.High, tc.Low, tc.Close, tc.Volume)

		// Stampa gli indicatori se disponibili
		fmt.Printf("  Indicatori Tecnici:\n")

		if tc.EMA20 != nil {
			fmt.Printf("    EMA20:  %.4f\n", *tc.EMA20)
		} else {
			fmt.Printf("    EMA20:  N/A (dati insufficienti)\n")
		}

		if tc.EMA60 != nil {
			fmt.Printf("    EMA60:  %.4f\n", *tc.EMA60)
		} else {
			fmt.Printf("    EMA60:  N/A (dati insufficienti)\n")
		}

		if tc.EMA223 != nil {
			fmt.Printf("    EMA223: %.4f\n", *tc.EMA223)
		} else {
			fmt.Printf("    EMA223: N/A (dati insufficienti)\n")
		}

		if tc.RSI14 != nil {
			fmt.Printf("    RSI14:  %.2f", *tc.RSI14)
			// Aggiungi interpretazione RSI
			rsiVal := *tc.RSI14
			if rsiVal > 70 {
				fmt.Printf(" (IPERCOMPRATO)")
			} else if rsiVal < 30 {
				fmt.Printf(" (IPERVENDUTO)")
			} else {
				fmt.Printf(" (NEUTRALE)")
			}
			fmt.Printf("\n")
		} else {
			fmt.Printf("    RSI14:  N/A (dati insufficienti)\n")
		}

		// Verifica se tutti gli indicatori sono disponibili
		if tc.HasAllIndicators() {
			fmt.Printf("  ✅ Tutti gli indicatori calcolati\n")
		} else {
			fmt.Printf("  ⚠️  Alcuni indicatori non disponibili\n")
		}
	}

	fmt.Printf("\n" + strings.Repeat("-", 80) + "\n")
	fmt.Printf("Test indicatori tecnici completato con successo!\n")

	// Statistiche finali
	validIndicators := 0
	for _, tc := range taCandlesticks {
		if tc.HasAllIndicators() {
			validIndicators++
		}
	}

	fmt.Printf("\nStatistiche:\n")
	fmt.Printf("- Candele totali processate: %d\n", len(taCandlesticks))
	fmt.Printf("- Candele con tutti gli indicatori: %d\n", validIndicators)
	fmt.Printf("- Percentuale di completezza: %.1f%%\n",
		float64(validIndicators)/float64(len(taCandlesticks))*100)

	return nil
}
