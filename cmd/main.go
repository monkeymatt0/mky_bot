package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cross-exchange-arbitrage/config"
	"cross-exchange-arbitrage/exchange"
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
	fmt.Printf("Avvio monitoraggio in tempo reale per %s su Bybit Perpetuals...\n", symbol)

	// Avvia il monitoraggio in una goroutine separata
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				priceData, err := bybitExchange.GetRealTimePrice(ctx, symbol)
				if err != nil {
					log.Printf("Errore nel recupero dati per %s: %v", symbol, err)
					time.Sleep(5 * time.Second)
					continue
				}

				fmt.Printf("\n=== AGGIORNAMENTO PREZZI %s ===\n", symbol)
				fmt.Printf("Timestamp: %s\n", priceData.Timestamp.Format("15:04:05.000"))
				fmt.Printf("PREZZO MEDIO: %.4f USDT\n", priceData.Price)
				fmt.Printf("BID: %.4f USDT (LIQUIDITA: %.4f USDT)\n", priceData.BidPrice, priceData.BidLiquidity)
				fmt.Printf("ASK: %.4f USDT (LIQUIDITA: %.4f USDT)\n", priceData.AskPrice, priceData.AskLiquidity)
				fmt.Printf("SPREAD: %.4f USDT (%.4f%%)\n",
					priceData.AskPrice-priceData.BidPrice,
					((priceData.AskPrice-priceData.BidPrice)/priceData.Price)*100)
				fmt.Println("================================")
			}
		}
	}()

	fmt.Println("Bot di arbitraggio inizializzato con successo!")
	fmt.Println("Premi Ctrl+C per terminare...")

	// Gestione graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Attendi il segnale di terminazione
	<-sigChan
	fmt.Println("\nTerminazione in corso...")
	cancel()

	// Attendi un po' per permettere alle goroutine di terminare
	time.Sleep(2 * time.Second)
	fmt.Println("Applicazione terminata.")
}
