package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cross-exchange-arbitrage/config"
	"cross-exchange-arbitrage/exchange"
	"cross-exchange-arbitrage/models"
	"cross-exchange-arbitrage/orderprocessor"
)

func main() {
	ctx := context.Background()

	fmt.Printf("🧪 TEST ORDINI BYBIT TESTNET\n")
	fmt.Printf("================================\n")

	// Carica configurazioni
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Errore nel caricamento della configurazione:", err)
	}

	// Verifica che le chiavi API siano configurate
	if cfg.Bybit.APIKey == "" || cfg.Bybit.SecretKey == "" {
		log.Fatal("❌ API Key e API Secret di Bybit non configurate nel file .env!")
	}

	fmt.Printf("✅ API Key configurata: %s...%s\n",
		cfg.Bybit.APIKey[:8],
		cfg.Bybit.APIKey[len(cfg.Bybit.APIKey)-8:])

	// Inizializza gli exchange per ottenere prezzi correnti
	bybitExchange := exchange.NewBybitExchange()
	defer bybitExchange.Close()

	// Test con BTCUSDT
	symbol := "BTCUSDT"

	fmt.Printf("\n📊 Recupero prezzo corrente di %s...\n", symbol)

	// Ottieni il prezzo corrente per calcolare i parametri dell'ordine
	priceData, err := bybitExchange.GetRealTimePrice(ctx, symbol)
	if err != nil {
		log.Printf("⚠️ Errore nel recupero del prezzo corrente: %v", err)
		log.Printf("📝 Userò prezzo di default per il test")
	}

	var currentPrice float64 = 50000.0 // Prezzo di fallback
	if priceData != nil {
		currentPrice = priceData.Price
		fmt.Printf("💰 Prezzo corrente %s: %.2f USDT\n", symbol, currentPrice)
	} else {
		fmt.Printf("💰 Usando prezzo di test: %.2f USDT\n", currentPrice)
	}

	// Inizializza il processor di ordini per TESTNET
	processor := orderprocessor.NewBybitTestnetOrderProcessor(cfg.Bybit.APIKey, cfg.Bybit.SecretKey)
	fmt.Printf("🧪 Usando Bybit TESTNET API: https://api-testnet.bybit.com\n")

	// Parametri per il test
	quantity := 0.001 // Quantità piccola per test (0.001 BTC)

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📈 TEST ORDINE LONG COMPLETO\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Calcola parametri per ordine LONG
	longTriggerPrice := currentPrice + 500 // Compra quando il prezzo sale di 500 USDT
	longStopLoss := currentPrice - 1000    // Stop loss a -1000 USDT dal prezzo corrente
	longTakeProfit := currentPrice + 2000  // Take profit a +2000 USDT dal prezzo corrente

	fmt.Printf("📋 Parametri Ordine LONG:\n")
	fmt.Printf("   Symbol: %s\n", symbol)
	fmt.Printf("   Quantity: %.6f BTC\n", quantity)
	fmt.Printf("   Trigger Price: %.2f USDT (attuale + 500)\n", longTriggerPrice)
	fmt.Printf("   Stop Loss: %.2f USDT (attuale - 1000)\n", longStopLoss)
	fmt.Printf("   Take Profit: %.2f USDT (attuale + 2000)\n", longTakeProfit)
	fmt.Printf("   💡 L'ordine si attiverà quando BTC raggiunge %.2f USDT\n", longTriggerPrice)

	// Piazza ordine LONG
	fmt.Printf("\n🚀 Piazzando ordine LONG...\n")
	longOrder, err := processor.PlaceLongOrder(
		ctx,
		symbol,
		longTriggerPrice, // trigger price
		quantity,         // quantity
		longStopLoss,     // stop loss
		longTakeProfit,   // take profit
	)

	if err != nil {
		fmt.Printf("❌ ERRORE ordine LONG: %v\n", err)
	} else {
		fmt.Printf("✅ ORDINE LONG PIAZZATO CON SUCCESSO!\n")
		printOrderDetails(longOrder, "LONG")
	}

	// Aspetta tra gli ordini per evitare rate limiting
	fmt.Printf("\n⏳ Pausa di 3 secondi per evitare rate limiting...\n")
	time.Sleep(3 * time.Second)

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📉 TEST ORDINE SHORT COMPLETO\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Calcola parametri per ordine SHORT
	shortTriggerPrice := currentPrice - 500 // Vendi quando il prezzo scende di 500 USDT
	shortStopLoss := currentPrice + 1000    // Stop loss a +1000 USDT dal prezzo corrente (per short)
	shortTakeProfit := currentPrice - 2000  // Take profit a -2000 USDT dal prezzo corrente (per short)

	fmt.Printf("📋 Parametri Ordine SHORT:\n")
	fmt.Printf("   Symbol: %s\n", symbol)
	fmt.Printf("   Quantity: %.6f BTC\n", quantity)
	fmt.Printf("   Trigger Price: %.2f USDT (attuale - 500)\n", shortTriggerPrice)
	fmt.Printf("   Stop Loss: %.2f USDT (attuale + 1000)\n", shortStopLoss)
	fmt.Printf("   Take Profit: %.2f USDT (attuale - 2000)\n", shortTakeProfit)
	fmt.Printf("   💡 L'ordine si attiverà quando BTC scende a %.2f USDT\n", shortTriggerPrice)

	// Piazza ordine SHORT
	fmt.Printf("\n🚀 Piazzando ordine SHORT...\n")
	shortOrder, err := processor.PlaceShortOrder(
		ctx,
		symbol,
		shortTriggerPrice, // trigger price
		quantity,          // quantity
		shortStopLoss,     // stop loss
		shortTakeProfit,   // take profit
	)

	if err != nil {
		fmt.Printf("❌ ERRORE ordine SHORT: %v\n", err)
	} else {
		fmt.Printf("✅ ORDINE SHORT PIAZZATO CON SUCCESSO!\n")
		printOrderDetails(shortOrder, "SHORT")
	}

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("🎯 RIEPILOGO TEST COMPLETATO\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	if longOrder != nil && longOrder.IsSuccess() {
		fmt.Printf("✅ Ordine LONG: SUCCESS (OrderID: %s)\n", longOrder.OrderID)
	} else {
		fmt.Printf("❌ Ordine LONG: FAILED\n")
	}

	if shortOrder != nil && shortOrder.IsSuccess() {
		fmt.Printf("✅ Ordine SHORT: SUCCESS (OrderID: %s)\n", shortOrder.OrderID)
	} else {
		fmt.Printf("❌ Ordine SHORT: FAILED\n")
	}

	fmt.Printf("\n📝 NOTA IMPORTANTE:\n")
	fmt.Printf("   - Questi sono ordini CONDIZIONALI (Stop Orders)\n")
	fmt.Printf("   - Si attiveranno solo quando il prezzo raggiunge il trigger\n")
	fmt.Printf("   - Puoi controllarli su: https://testnet.bybit.com/\n")
	fmt.Printf("   - Gli ordini rimangono attivi fino a trigger o cancellazione\n")

	fmt.Printf("\n🏁 Test completato!\n")
}

// printOrderDetails stampa i dettagli completi di un ordine
func printOrderDetails(order *models.OrderResponse, orderType string) {
	fmt.Printf("\n📄 Dettagli Ordine %s:\n", orderType)
	fmt.Printf("   ┌─ Order ID: %s\n", order.OrderID)
	fmt.Printf("   ├─ Order Link ID: %s\n", order.OrderLinkID)
	fmt.Printf("   ├─ Symbol: %s\n", order.Symbol)
	fmt.Printf("   ├─ Side: %s\n", order.Side)
	fmt.Printf("   ├─ Order Type: %s\n", order.OrderType)
	fmt.Printf("   ├─ Status: %s\n", order.Status)
	fmt.Printf("   ├─ Quantity: %.6f\n", order.Quantity)

	if order.Price > 0 {
		fmt.Printf("   ├─ Price: %.2f USDT\n", order.Price)
	}

	if order.TriggerPrice > 0 {
		fmt.Printf("   ├─ Trigger Price: %.2f USDT\n", order.TriggerPrice)
	}

	if order.StopLoss > 0 {
		fmt.Printf("   ├─ Stop Loss: %.2f USDT\n", order.StopLoss)
	}

	if order.TakeProfit > 0 {
		fmt.Printf("   ├─ Take Profit: %.2f USDT\n", order.TakeProfit)
	}

	fmt.Printf("   ├─ Created: %s\n", order.CreatedTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   └─ Success: %t\n", order.IsSuccess())

	if !order.IsSuccess() && order.ErrorMessage != "" {
		fmt.Printf("\n   ⚠️ Error Details:\n")
		fmt.Printf("      Code: %s\n", order.ErrorCode)
		fmt.Printf("      Message: %s\n", order.ErrorMessage)
	}
}
