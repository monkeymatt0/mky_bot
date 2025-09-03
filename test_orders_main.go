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
	bybitExchange := exchange.NewBybitExchange(true)
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
	longTriggerPrice := currentPrice + 50_000  // Compra quando il prezzo sale di 500 USDT
	longLimitPrice := longTriggerPrice * 1.002 // Prezzo limite 0.2% sopra il trigger
	longStopLoss := currentPrice - 51_000      // Stop loss a -1000 USDT dal prezzo corrente
	longTakeProfit := currentPrice + 52_000    // Take profit a +2000 USDT dal prezzo corrente

	fmt.Printf("📋 Parametri Ordine LONG (Stop-Limit):\n")
	fmt.Printf("   Symbol: %s\n", symbol)
	fmt.Printf("   Quantity: %.6f BTC\n", quantity)
	fmt.Printf("   Trigger Price: %.2f USDT (attuale + 500)\n", longTriggerPrice)
	fmt.Printf("   Limit Price: %.2f USDT (trigger + 0.2%%)\n", longLimitPrice)
	fmt.Printf("   Stop Loss: %.2f USDT (attuale - 1000)\n", longStopLoss)
	fmt.Printf("   Take Profit: %.2f USDT (attuale + 2000)\n", longTakeProfit)
	fmt.Printf("   💡 L'ordine si attiverà quando BTC raggiunge %.2f USDT\n", longTriggerPrice)
	fmt.Printf("   🎯 Poi comprerà a massimo %.2f USDT (prezzo fisso)\n", longLimitPrice)

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
	shortTriggerPrice := currentPrice - 500      // Vendi quando il prezzo scende di 500 USDT
	shortLimitPrice := shortTriggerPrice * 0.998 // Prezzo limite 0.2% sotto il trigger
	shortStopLoss := currentPrice + 1000         // Stop loss a +1000 USDT dal prezzo corrente (per short)
	shortTakeProfit := currentPrice - 2000       // Take profit a -2000 USDT dal prezzo corrente (per short)

	fmt.Printf("📋 Parametri Ordine SHORT (Stop-Limit):\n")
	fmt.Printf("   Symbol: %s\n", symbol)
	fmt.Printf("   Quantity: %.6f BTC\n", quantity)
	fmt.Printf("   Trigger Price: %.2f USDT (attuale - 500)\n", shortTriggerPrice)
	fmt.Printf("   Limit Price: %.2f USDT (trigger - 0.2%%)\n", shortLimitPrice)
	fmt.Printf("   Stop Loss: %.2f USDT (attuale + 1000)\n", shortStopLoss)
	fmt.Printf("   Take Profit: %.2f USDT (attuale - 2000)\n", shortTakeProfit)
	fmt.Printf("   💡 L'ordine si attiverà quando BTC scende a %.2f USDT\n", shortTriggerPrice)
	fmt.Printf("   🎯 Poi venderà a minimo %.2f USDT (prezzo fisso)\n", shortLimitPrice)

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

	// Test verifica stato ordini se ne abbiamo creati con successo
	if (longOrder != nil && longOrder.IsSuccess()) || (shortOrder != nil && shortOrder.IsSuccess()) {
		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("🔍 TEST VERIFICA STATO ORDINI\n")
		fmt.Printf(strings.Repeat("=", 60) + "\n")

		// Aspetta 2 secondi prima di verificare lo stato
		fmt.Printf("⏳ Pausa di 2 secondi prima della verifica stato...\n")
		time.Sleep(2 * time.Second)

		// Verifica stato ordine LONG
		if longOrder != nil && longOrder.IsSuccess() {
			fmt.Printf("\n📈 Verifica Stato Ordine LONG:\n")
			fmt.Printf("--------------------------------\n")
			fmt.Printf("🎯 OrderID: %s\n", longOrder.OrderID)

			orderStatus, err := processor.GetOrderStatus(ctx, symbol, longOrder.OrderID)
			if err != nil {
				fmt.Printf("❌ Errore nel recupero stato LONG: %v\n", err)
			} else {
				fmt.Printf("✅ STATO ORDINE LONG RECUPERATO!\n")
				fmt.Printf("   ├─ Status: %s\n", orderStatus.Status)
				fmt.Printf("   ├─ Order Type: %s\n", orderStatus.OrderType)
				fmt.Printf("   ├─ Side: %s\n", orderStatus.Side)
				fmt.Printf("   ├─ Price: %.2f USDT\n", orderStatus.Price)
				fmt.Printf("   ├─ Quantity: %.6f BTC\n", orderStatus.Quantity)
				fmt.Printf("   ├─ Created: %s\n", orderStatus.CreatedTime.Format("2006-01-02 15:04:05"))
				fmt.Printf("   └─ Updated: %s\n", orderStatus.UpdatedTime.Format("2006-01-02 15:04:05"))

				// Verifica se può essere aggiornato
				if processor.CanBeUpdated(orderStatus.Status) {
					fmt.Printf("   ✅ L'ordine LONG può essere aggiornato\n")
				} else {
					fmt.Printf("   ⚠️ L'ordine LONG NON può essere aggiornato (Status: %s)\n", orderStatus.Status)
					fmt.Printf("   📝 Spiegazione: Gli ordini 'Untriggered' non hanno ancora posizioni aperte\n")
				}
			}
		}

		// Verifica stato ordine SHORT
		if shortOrder != nil && shortOrder.IsSuccess() {
			fmt.Printf("\n📉 Verifica Stato Ordine SHORT:\n")
			fmt.Printf("---------------------------------\n")
			fmt.Printf("🎯 OrderID: %s\n", shortOrder.OrderID)

			// Pausa tra le verifiche per evitare rate limiting
			fmt.Printf("⏳ Pausa di 1 secondo prima della verifica SHORT...\n")
			time.Sleep(1 * time.Second)

			orderStatus, err := processor.GetOrderStatus(ctx, symbol, shortOrder.OrderID)
			if err != nil {
				fmt.Printf("❌ Errore nel recupero stato SHORT: %v\n", err)
			} else {
				fmt.Printf("✅ STATO ORDINE SHORT RECUPERATO!\n")
				fmt.Printf("   ├─ Status: %s\n", orderStatus.Status)
				fmt.Printf("   ├─ Order Type: %s\n", orderStatus.OrderType)
				fmt.Printf("   ├─ Side: %s\n", orderStatus.Side)
				fmt.Printf("   ├─ Price: %.2f USDT\n", orderStatus.Price)
				fmt.Printf("   ├─ Quantity: %.6f BTC\n", orderStatus.Quantity)
				fmt.Printf("   ├─ Created: %s\n", orderStatus.CreatedTime.Format("2006-01-02 15:04:05"))
				fmt.Printf("   └─ Updated: %s\n", orderStatus.UpdatedTime.Format("2006-01-02 15:04:05"))

				// Verifica se può essere aggiornato
				if processor.CanBeUpdated(orderStatus.Status) {
					fmt.Printf("   ✅ L'ordine SHORT può essere aggiornato\n")
				} else {
					fmt.Printf("   ⚠️ L'ordine SHORT NON può essere aggiornato (Status: %s)\n", orderStatus.Status)
					fmt.Printf("   📝 Spiegazione: Gli ordini 'Untriggered' non hanno ancora posizioni aperte\n")
				}
			}
		}

		fmt.Printf("\n🎯 Verifica stati completata!\n")
		fmt.Printf("📝 Ora sappiamo lo stato esatto degli ordini prima dell'aggiornamento\n")
		fmt.Printf("💡 Questo spiega perché gli aggiornamenti SL/TP falliranno con ordini condizionali\n")
	}

	// Test di aggiornamento ordini se ne abbiamo creati con successo
	if (longOrder != nil && longOrder.IsSuccess()) || (shortOrder != nil && shortOrder.IsSuccess()) {
		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("🔄 TEST AGGIORNAMENTO ORDINI\n")
		fmt.Printf(strings.Repeat("=", 60) + "\n")

		// Aspetta 3 secondi prima di aggiornare
		fmt.Printf("⏳ Pausa di 3 secondi prima degli aggiornamenti...\n")
		time.Sleep(3 * time.Second)

		// Test aggiornamento ordine LONG
		if longOrder != nil && longOrder.IsSuccess() {
			fmt.Printf("\n📈 Aggiornamento Ordine LONG:\n")
			fmt.Printf("-----------------------------\n")
			fmt.Printf("🎯 OrderID: %s\n", longOrder.OrderID)

			// Nuovi valori per ordine LONG
			newLongStopLoss := currentPrice - 2000.0   // Stop loss più conservativo
			newLongTakeProfit := currentPrice + 3000.0 // Take profit più ambizioso

			fmt.Printf("📊 Valori originali:\n")
			fmt.Printf("   Stop Loss: %.2f USDT\n", longStopLoss)
			fmt.Printf("   Take Profit: %.2f USDT\n", longTakeProfit)
			fmt.Printf("📊 Nuovi valori:\n")
			fmt.Printf("   Stop Loss: %.2f USDT (più conservativo)\n", newLongStopLoss)
			fmt.Printf("   Take Profit: %.2f USDT (più ambizioso)\n", newLongTakeProfit)

			updateParamsLong := orderprocessor.UpdateOrderParams{
				Symbol:      symbol,
				StopLoss:    &newLongStopLoss,
				TakeProfit:  &newLongTakeProfit,
				PositionIdx: 0, // One-way mode
			}

			updateRespLong, err := processor.UpdateOrder(ctx, updateParamsLong)
			if err != nil {
				fmt.Printf("❌ Errore aggiornamento ordine LONG: %v\n", err)
			} else {
				fmt.Printf("✅ ORDINE LONG AGGIORNATO CON SUCCESSO!\n")
				fmt.Printf("   ├─ Status: %s\n", updateRespLong.Status)
				fmt.Printf("   ├─ Success: %t\n", updateRespLong.IsSuccess())
				fmt.Printf("   ├─ New Stop Loss: %.2f USDT\n", updateRespLong.StopLoss)
				fmt.Printf("   └─ New Take Profit: %.2f USDT\n", updateRespLong.TakeProfit)

				if !updateRespLong.IsSuccess() {
					fmt.Printf("   ⚠️ Errore: %s\n", updateRespLong.ErrorMessage)
				}
			}
		}

		// Test aggiornamento ordine SHORT
		if shortOrder != nil && shortOrder.IsSuccess() {
			fmt.Printf("\n📉 Aggiornamento Ordine SHORT:\n")
			fmt.Printf("------------------------------\n")
			fmt.Printf("🎯 OrderID: %s\n", shortOrder.OrderID)

			// Pausa tra gli aggiornamenti per evitare rate limiting
			fmt.Printf("⏳ Pausa di 2 secondi prima dell'aggiornamento SHORT...\n")
			time.Sleep(2 * time.Second)

			// Nuovi valori per ordine SHORT
			newShortStopLoss := currentPrice + 1500.0   // Stop loss più conservativo
			newShortTakeProfit := currentPrice - 3000.0 // Take profit più ambizioso

			fmt.Printf("📊 Valori originali:\n")
			fmt.Printf("   Stop Loss: %.2f USDT\n", shortStopLoss)
			fmt.Printf("   Take Profit: %.2f USDT\n", shortTakeProfit)
			fmt.Printf("📊 Nuovi valori:\n")
			fmt.Printf("   Stop Loss: %.2f USDT (più conservativo)\n", newShortStopLoss)
			fmt.Printf("   Take Profit: %.2f USDT (più ambizioso)\n", newShortTakeProfit)

			updateParamsShort := orderprocessor.UpdateOrderParams{
				Symbol:      symbol,
				StopLoss:    &newShortStopLoss,
				TakeProfit:  &newShortTakeProfit,
				PositionIdx: 0, // One-way mode
			}

			updateRespShort, err := processor.UpdateOrder(ctx, updateParamsShort)
			if err != nil {
				fmt.Printf("❌ Errore aggiornamento ordine SHORT: %v\n", err)
			} else {
				fmt.Printf("✅ ORDINE SHORT AGGIORNATO CON SUCCESSO!\n")
				fmt.Printf("   ├─ Status: %s\n", updateRespShort.Status)
				fmt.Printf("   ├─ Success: %t\n", updateRespShort.IsSuccess())
				fmt.Printf("   ├─ New Stop Loss: %.2f USDT\n", updateRespShort.StopLoss)
				fmt.Printf("   └─ New Take Profit: %.2f USDT\n", updateRespShort.TakeProfit)

				if !updateRespShort.IsSuccess() {
					fmt.Printf("   ⚠️ Errore: %s\n", updateRespShort.ErrorMessage)
				}
			}
		}

		fmt.Printf("\n🎯 Aggiornamenti completati!\n")
		fmt.Printf("📝 Gli ordini ora hanno Stop Loss e Take Profit aggiornati\n")
	}

	// Test di cancellazione ordini se ne abbiamo creati con successo
	ordersToCancel := []struct {
		order     *models.OrderResponse
		orderType string
	}{}

	if longOrder != nil && longOrder.IsSuccess() {
		ordersToCancel = append(ordersToCancel, struct {
			order     *models.OrderResponse
			orderType string
		}{longOrder, "LONG"})
	}

	if shortOrder != nil && shortOrder.IsSuccess() {
		ordersToCancel = append(ordersToCancel, struct {
			order     *models.OrderResponse
			orderType string
		}{shortOrder, "SHORT"})
	}

	if len(ordersToCancel) > 0 {
		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("🗑️ TEST CANCELLAZIONE ORDINI\n")
		fmt.Printf(strings.Repeat("=", 60) + "\n")

		fmt.Printf("🎯 Ordini da cancellare: %d\n", len(ordersToCancel))

		// Aspetta 2 secondi prima di iniziare le cancellazioni
		fmt.Printf("⏳ Pausa di 2 secondi prima delle cancellazioni...\n")
		time.Sleep(2 * time.Second)

		// Cancella tutti gli ordini creati
		for i, orderInfo := range ordersToCancel {
			fmt.Printf("\n📋 Cancellazione ordine %s (%d/%d):\n",
				orderInfo.orderType, i+1, len(ordersToCancel))
			fmt.Printf("   OrderID: %s\n", orderInfo.order.OrderID)

			cancelResp, err := processor.DeleteOrder(ctx, symbol, orderInfo.order.OrderID)
			if err != nil {
				fmt.Printf("❌ Errore nella cancellazione ordine %s: %v\n", orderInfo.orderType, err)
			} else {
				fmt.Printf("✅ ORDINE %s CANCELLATO CON SUCCESSO!\n", orderInfo.orderType)
				fmt.Printf("   ├─ OrderID: %s\n", cancelResp.OrderID)
				fmt.Printf("   ├─ Status: %s\n", cancelResp.Status)
				fmt.Printf("   └─ Success: %t\n", cancelResp.IsSuccess())

				if !cancelResp.IsSuccess() {
					fmt.Printf("   ⚠️ Errore: %s\n", cancelResp.ErrorMessage)
				}
			}

			// Pausa tra le cancellazioni per evitare rate limiting
			if i < len(ordersToCancel)-1 {
				fmt.Printf("   ⏳ Pausa di 1 secondo prima del prossimo ordine...\n")
				time.Sleep(1 * time.Second)
			}
		}

		fmt.Printf("\n🎯 Riepilogo cancellazioni completato!\n")
	} else {
		fmt.Printf("\n⚠️ Nessun ordine da cancellare (nessun ordine creato con successo)\n")
	}

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
