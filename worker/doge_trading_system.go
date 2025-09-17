package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"time"

	"cross-exchange-arbitrage/config"
	"cross-exchange-arbitrage/database"
	"cross-exchange-arbitrage/exchange"
	"cross-exchange-arbitrage/models"
	"cross-exchange-arbitrage/orderprocessor"
	"cross-exchange-arbitrage/repositories"
	"cross-exchange-arbitrage/services"

	"github.com/markcheno/go-talib"
	"gorm.io/gorm"
)

// DogeTradingSystemWorker rappresenta il worker per il sistema di trading DOGE
type DogeTradingSystemWorker struct {
	ctx            context.Context
	cancel         context.CancelFunc
	exchange       exchange.Exchange
	orderProcessor orderprocessor.OrderProcessor
	db             *gorm.DB
	orderService   *services.OrderService
	orderPlaced    bool // Flag per indicare se c'è un ordine già piazzato
}

// NewDogeTradingSystemWorker crea una nuova istanza del worker
func NewDogeTradingSystemWorker() *DogeTradingSystemWorker {
	ctx, cancel := context.WithCancel(context.Background())

	// Carica la configurazione
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Errore nel caricamento della configurazione: %v", err)
		// Usa valori di default per testnet
		cfg = &config.Config{
			Bybit: config.BybitConfig{
				APIKey:    os.Getenv("BYBIT_API_KEY"),
				SecretKey: os.Getenv("BYBIT_SECRET_KEY"),
			},
		}
	}

	// Inizializza database
	log.Println("Inizializzando database per DOGE Trading System...")
	db, err := database.InitializeDatabaseWithData(database.DefaultConfig())
	if err != nil {
		log.Fatalf("ERRORE CRITICO: Impossibile inizializzare database: %v", err)
	}

	// Crea repository manager e order service
	repoManager := repositories.NewRepositoryManager(db)
	orderService := services.NewOrderService(repoManager)

	// Crea il processor per gli ordini
	var orderProcessor orderprocessor.OrderProcessor
	if cfg.Bybit.APIKey != "" && cfg.Bybit.SecretKey != "" {
		orderProcessor = orderprocessor.NewBybitOrderProcessor(cfg.Bybit.APIKey, cfg.Bybit.SecretKey)
	} else {
		log.Println("ATTENZIONE: Credenziali API Bybit non configurate, ordini non funzioneranno")
		orderProcessor = nil
	}

	return &DogeTradingSystemWorker{
		ctx:            ctx,
		cancel:         cancel,
		exchange:       exchange.NewBybitExchange(false), // false = usa produzione, true = usa testnet
		orderProcessor: orderProcessor,
		db:             db,
		orderService:   orderService,
	}
}

// ExecuteTradingCycle esegue un ciclo completo di trading (metodo pubblico per test)
func (w *DogeTradingSystemWorker) ExecuteTradingCycle() {
	w.executeTradingCycle()
}

// executeTradingCycle esegue un ciclo completo di trading
func (w *DogeTradingSystemWorker) executeTradingCycle() {
	log.Println("Executing DOGE Trading Cycle...")

	// ========================================
	// FASE 0: Controllo flag orderPlaced
	// ========================================
	log.Println("Phase 0: Checking orderPlaced status...")
	orderPlaced, err := w.isPostionActive("DOGEUSDT") // Questo aggiorna anche il DB nel caso siano trovate delle posizioni
	if err != nil {
		log.Printf("Errore nel controllo orderPlaced: %v", err)
		return
	}

	// Il monitoraggio dell'ordine verrà fatto da un altro servizio che si occuperà
	// solo di monitorare l'ordine.

	w.orderPlaced = orderPlaced

	// Se l'ordine + piazzato allora non faccio nulla
	if w.orderPlaced {
		log.Println("🔄 orderPlaced=true - Bypass del ciclo di trading, riprova tra 5 minuti")
		return
	}

	// ========================================
	// FASE 1: Fetch delle ultime 1000 candele
	// ========================================
	log.Println("Phase 1: Fetching last 1000 candles...")
	candleResponse := w.fetchLast1000Candles()
	if candleResponse == nil {
		log.Println("Failed to fetch candles, skipping cycle")
		return
	}

	// Estrai le ultime 5 candele chiuse (escludendo quella attualmente aperta e l'ultima chiusa)
	last40Candles, wall, support, err := w.extractCandlesForChecks(candleResponse.Candles)
	currentClosedCandle := candleResponse.Candles[len(candleResponse.Candles)-2] // Ultima candela chiusa ovvero la penultima
	if err != nil {
		log.Printf("Error extracting candles for checks: %v", err)
		return
	}

	// 3 Controllo rottura muro delle ultime 40 candele precedenti con chiusura sopra il muro o sotto la resistenza
	wallBreak, supportBreak := w.checkWallAndSupportBreak(currentClosedCandle, last40Candles, wall, support)

	// Se rompe il muro allora faccio i check sul volume per le candele verdi
	if wallBreak { // Rottura del muro delle 5 candele precedenti

		log.Println("All conditions met! Proceeding with LONG order...")
		// 4 Calcolo media volume candele verdi delle ultime 5 candele verdi
		log.Println("Resistance broken! -----> Calculating green candles average volume...")
		greenCandlesVolumeTotAvg := w.calculateGreenCandlesAverageVolume(candleResponse.Candles)
		// Adesso prendo il volume dell'ultima candela chiusa che ha chiuso sopra il muro
		lastCandleVolume := currentClosedCandle.Volume

		log.Printf("Last candle volume: %.2f", lastCandleVolume)
		log.Printf("Green candles averageVolumeTOT: %.2f", greenCandlesVolumeTotAvg)

		// Se il volume dell'ultima candela è maggiore del rapporto
		if greenCandlesVolumeTotAvg > 0.6 && currentClosedCandle.Volume > greenCandlesVolumeTotAvg*1.2 {
			// In questo caso tutti i check sono passati quindi vuol dire che troviamo
			// di fronte ad una potenziale opportunità di trading
			log.Println("All conditions met! Proceeding with LONG order...")
			// ========================================
			// FASE 3.1: Piazzamento ordine LONG
			// ========================================
			price := currentClosedCandle.Close
			orderID := w.placeLongOrder(price)
			if orderID == "" {
				log.Println("Failed to place LONG order")
				time.Sleep(1 * time.Second)
				orderID = w.placeLongOrder(price)
				if orderID == "" {
					log.Println("Failed to place LONG order second time")
					log.Println("Trying last time")
					time.Sleep(1 * time.Second)
					orderID = w.placeLongOrder(price)
					if orderID == "" {
						log.Println("Failed to place LONG order third time")
						return
					}
					return
				}
				return
			}
		}
	} else if supportBreak { // Rottura del supporto delle 5 candele precedenti, qui calcolo il volume per le candele rosse

		log.Println("All conditions met! Proceeding with SHORT order...")

		// 5 Calcolo media volume candele rosse delle ultime 5 candele rosse
		log.Println("Support broken! -----> Calculating red candles average volume...")
		redCandlesVolumeTotAvg := w.calculateRedCandlesAverageVolume(candleResponse.Candles)
		// Adesso prendo il volume dell'ultima candela che ha chiuso sotto il supporto
		lastCandleVolume := currentClosedCandle.Volume

		log.Printf("Last candle volume: %.2f", lastCandleVolume)
		log.Printf("Green candles averageVolumeTOT: %.2f", redCandlesVolumeTotAvg)

		if redCandlesVolumeTotAvg > 0.6 && currentClosedCandle.Volume > redCandlesVolumeTotAvg*1.2 {
			// In questo caso tutti i check sono passati quindi vuol dire che troviamo
			// di fronte ad una potenziale opportunità di trading
			log.Println("All conditions met! Proceeding with SHORT order...")

			// ========================================
			// FASE 3.1: Piazzamento ordine SHORT
			// ========================================
			price := currentClosedCandle.Close
			orderID := w.placeShortOrder(price) // Da implementare
			if orderID == "" {
				log.Println("Failed to place SHORT order")
				time.Sleep(1 * time.Second)
				orderID = w.placeShortOrder(price)
				if orderID == "" {
					log.Println("Failed to place SHORT order second time")
					log.Println("Trying last time")
					time.Sleep(1 * time.Second)
					orderID = w.placeShortOrder(price)
					if orderID == "" {
						log.Println("Failed to place SHORT order third time")
						return
					}
					return
				}
				return
			}
		}
	} else {
		log.Println("Trading conditions not met, skipping order placement")
	}
}

// GetName implementa l'interfaccia Worker
func (w *DogeTradingSystemWorker) GetName() string {
	return "DOGE Trading System Worker"
}

// Start avvia il worker (DEPRECATO - usa il nuovo sistema cron)
// Questo metodo è mantenuto per compatibilità ma non dovrebbe essere usato
func (w *DogeTradingSystemWorker) Start() {
	log.Println("⚠️  ATTENZIONE: Metodo Start() deprecato!")
	log.Println("⚠️  Usa il nuovo sistema worker con cron: worker.StartWorkerSystem()")
	log.Println("⚠️  Questo metodo non fa nulla per evitare conflitti con il cron")
}

// Stop ferma il worker
func (w *DogeTradingSystemWorker) Stop() {
	log.Println("Stopping DOGE Trading System Worker...")
	w.cancel()

	// Chiudi connessione database
	if w.db != nil {
		if err := database.Close(w.db); err != nil {
			log.Printf("Errore chiusura database: %v", err)
		}
	}
}

// mapBybitStatusToOrderStatusID mappa lo stato Bybit al OrderStatusID del database
func (w *DogeTradingSystemWorker) mapBybitStatusToOrderStatusID(bybitStatus string) (uint, error) {
	// Mappa stati Bybit comuni
	statusMap := map[string]string{
		"New":                     "New",
		"PartiallyFilled":         "PartiallyFilled",
		"Filled":                  "Filled",
		"Cancelled":               "Cancelled",
		"Rejected":                "Rejected",
		"Untriggered":             "Untriggered",
		"Triggered":               "Triggered",
		"Deactivated":             "Deactivated",
		"PartiallyFilledCanceled": "PartiallyFilledCanceled",
	}

	// Cerca lo stato mappato
	mappedStatus, exists := statusMap[bybitStatus]
	if !exists {
		// Se non trovato, usa "New" come default
		mappedStatus = "New"
		log.Printf("Stato Bybit '%s' non mappato, uso 'New' come default", bybitStatus)
	}

	// Recupera l'ID dal database
	repoManager := repositories.NewRepositoryManager(w.db)
	status, err := repoManager.OrderStatus().GetByStatusName(w.ctx, mappedStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to get order status '%s': %w", mappedStatus, err)
	}

	return status.ID, nil
}

// createOrderFromBybitResponse crea un Order dal OrderResponse
func (w *DogeTradingSystemWorker) createOrderFromBybitResponse(
	bybitResponse *models.OrderResponse,
	triggerPrice, quantity, takeProfit, stopLoss float64,
) (*models.Order, error) {
	// Mappa lo stato Bybit
	orderStatusID, err := w.mapBybitStatusToOrderStatusID(string(bybitResponse.Status))
	if err != nil {
		return nil, fmt.Errorf("failed to map Bybit status: %w", err)
	}

	// Crea l'ordine
	order := &models.Order{
		OrderID:         bybitResponse.OrderID,
		Symbol:          "DOGEUSDT",
		Side:            models.OrderSideTypeBuy, // Sempre Buy per ordini LONG
		OrderPrice:      triggerPrice,
		Quantity:        quantity,
		TakeProfitPrice: &takeProfit,
		StopLossPrice:   &stopLoss,
		OrderStatusID:   orderStatusID,
		Result:          models.OrderResultPending,
		PnL:             0.0,
		PnLPercentage:   0.0,
	}

	return order, nil
}

// saveOrderToDatabase salva un ordine nel database
func (w *DogeTradingSystemWorker) saveOrderToDatabase(order *models.Order) error {
	if w.orderService == nil {
		return fmt.Errorf("order service not initialized")
	}

	err := w.orderService.CreateOrder(w.ctx, order)
	if err != nil {
		return fmt.Errorf("failed to save order to database: %w", err)
	}

	log.Printf("✅ Ordine salvato nel database: %s", order.OrderID)
	return nil
}

// CalculateMaxQuantity calcola la quantità massima basata su prezzo e saldo disponibile (metodo pubblico per test)
func (w *DogeTradingSystemWorker) CalculateMaxQuantity(price float64) float64 {
	return w.calculateMaxQuantity(price)
}

// GetUSDTBalance recupera il saldo USDT dal wallet (metodo pubblico per test)
func (w *DogeTradingSystemWorker) GetUSDTBalance() (float64, error) {
	balance, err := w.orderProcessor.GetUSDTBalance(w.ctx)
	if err != nil {
		return 0, err
	}
	return balance / 100, nil
}

// ========================================
// FASE 1: Fetch delle ultime 100 candele
// ========================================

// fetchLast1000Candles recupera le ultime 1000 candele
func (w *DogeTradingSystemWorker) fetchLast1000Candles() *models.CandleResponse {
	log.Println("Fetching last 1000 candles for DOGEUSDT...")

	// Fetch delle ultime 1000 candele per DOGEUSDT con timeframe 5m
	candleResponse, err := w.exchange.FetchLastCandles(
		w.ctx,
		"DOGEUSDT",
		models.DerivativesMarket, // Usa il mercato derivatives come da esempio nel progetto
		models.Timeframe1m,       // Timeframe 1 minut0
		1000,                     // Limite di 1000 candele
	)

	slices.Reverse(candleResponse.Candles) // Reverse dell'array in place, il che significa che adesso ho le candele ordine in maniera cronologica inversa (Dalla più recente alla più vecchia)

	if err != nil {
		log.Printf("Error fetching candles: %v", err)
		return nil
	}

	if candleResponse == nil {
		log.Println("No candles received from exchange")
		return nil
	}

	log.Printf("Successfully fetched %d candles for DOGEUSDT", len(candleResponse.Candles))
	return candleResponse
}

// ========================================
// FASE 2: Calcolo degli indicatori tecnici
// ========================================

// calculateTechnicalIndicators calcola gli indicatori tecnici necessari
func (w *DogeTradingSystemWorker) calculateTechnicalIndicators(candleResponse *models.CandleResponse) []*models.TACandlestick {
	log.Printf("Calculating technical indicators for %d candles...", len(candleResponse.Candles))

	// Verifica che ci siano abbastanza candele per calcolare l'RSI (minimo 14)
	if len(candleResponse.Candles) < 14 {
		log.Printf("Not enough candles for RSI calculation. Need at least 14, got %d", len(candleResponse.Candles))
		return nil
	}

	// Estrai i prezzi di chiusura per il calcolo dell'RSI
	closePrices := make([]float64, len(candleResponse.Candles))
	for i, candle := range candleResponse.Candles {
		closePrices[i] = candle.Close
	}

	// Calcola l'RSI con periodo 14
	rsiValues := talib.Rsi(closePrices, 14)

	// Crea le TACandlestick con i dati OHLCV e l'RSI calcolato
	taCandlesticks := make([]*models.TACandlestick, len(candleResponse.Candles))

	for i, candle := range candleResponse.Candles {
		taCandlestick := models.NewTACandlestickFromCandle(candle)

		// Imposta l'RSI14 se disponibile (i primi 13 valori saranno NaN)
		if i >= 13 && i < len(rsiValues) {
			taCandlestick.RSI14 = &rsiValues[i]
		}

		taCandlesticks[i] = taCandlestick
	}

	// Conta quante candele hanno l'RSI calcolato
	validRSICount := 0
	for _, taCandle := range taCandlesticks {
		if taCandle.RSI14 != nil {
			validRSICount++
		}
	}

	log.Printf("RSI calculation completed. %d candles have valid RSI values", validRSICount)
	return taCandlesticks
}

// ========================================
// FASE 3: Controlli per condizioni di trading
// ========================================

// extractCandlesForChecks estrae le ultime 40 candele al momento
func (w *DogeTradingSystemWorker) extractCandlesForChecks(taCandlesticks []models.Candle) ([]models.Candle, float64, float64, error) {
	// Le candele sono già in ordine cronologico (dalla più vecchia alla più recente)
	// La candela più recente è quella attualmente aperta, quindi la escludiamo

	if len(taCandlesticks) < 72 {
		return nil, 0.0, 0.0, fmt.Errorf("not enough candles for checks. Need at least 5, got %d", len(taCandlesticks))
	}

	// Estrai le ultime 5 candele chiuse (escludendo quella attualmente aperta)
	last72Candles := taCandlesticks[len(taCandlesticks)-74 : len(taCandlesticks)-2]

	// Prendo il massimo high delle ultime 5 candele chiuse
	last72CandlesWall := 0.0
	last72CandlesSupport := math.MaxFloat64 // Questo sarà il massimo numero quindi all'inizio sarà sempre il piu grande

	// Qui vado a fare una ricerca del massimo e del minimo in contemporanea
	for _, candle := range last72Candles {
		if candle.High > last72CandlesWall {
			last72CandlesWall = candle.High
		}
		if candle.Low < last72CandlesSupport {
			last72CandlesSupport = candle.Low
		}
	}

	log.Printf("Extracted %d last three candles and calculated wall: %.6f and support: %.6f", len(last72Candles), last72CandlesWall, last72CandlesSupport)

	return last72Candles, last72CandlesWall, last72CandlesSupport, nil
}

// Verifica se il muro è stato rotto o se il supporto è stato rotto
// Tuttavia questo check per essere valido c'è bisogno che l'ultima candela abbia il prezzo di chiusura
// sopra la resistenza o sotto il supporto.

// Se e solo se il prezzo chiudo nel modo giusto questa funzione ritornerò true per il muro o per il supporto
func (w *DogeTradingSystemWorker) checkWallAndSupportBreak(currentClosedCandle models.Candle, lastFiveCandles []models.Candle, wall float64, support float64) (bool, bool) {
	log.Printf("Checking wall break...")

	if len(lastFiveCandles) < 72 {
		log.Println("Not enough candles for wall break check")
		return false, false
	}

	wallBreak := false
	supportBreak := false
	lastCandle := currentClosedCandle

	// Calcola il muro (massimo high delle 5 candele del muro)
	if lastCandle.Close > wall {
		wallBreak = true
		log.Printf("Wall broken properly: %.6f", wall)
	} else if lastCandle.Close < support {
		supportBreak = true
		log.Printf("Support broken properly: %.6f", support)
	}

	return wallBreak, supportBreak
}

// ========================================
// FASE 3.1: Gestione ordini
// ========================================

// placeLongOrder piazza un ordine LONG
func (w *DogeTradingSystemWorker) placeLongOrder(currentPrice float64) string {
	log.Println("Placing LONG order...")

	// Verifica che il processor sia disponibile
	if w.orderProcessor == nil {
		log.Println("ERRORE: OrderProcessor non configurato, impossibile piazzare ordine")
		return ""
	}

	// Simbolo per DOGE
	symbol := "DOGEUSDT"

	// Calcola il prezzo di trigger basato sull'ultima candela
	triggerPrice := currentPrice
	if triggerPrice <= 0 {
		log.Println("ERRORE: Impossibile calcolare il prezzo di trigger")
		return ""
	}

	// Calcola la quantità massima basata sul saldo disponibile
	quantity := w.calculateMaxQuantity(triggerPrice)
	if quantity <= 0 {
		log.Println("ERRORE: Impossibile calcolare la quantità")
		return ""
	}

	longTriggerPrice := currentPrice
	takeProfit := w.calculateLongTakeProfit(currentPrice, 0.03)
	stopLoss := w.calculateLongStopLoss(currentPrice, 0.008)

	log.Printf("Parametri ordine LONG:")
	log.Printf("  Symbol: %s", symbol)
	log.Printf("  Trigger Price: $%.6f", triggerPrice)
	log.Printf("  Quantity: %.2f", quantity)
	log.Printf("  Stop Loss: $%.6f (%.3f%%)", stopLoss, 0.4)
	log.Printf("  Take Profit: $%.6f (%.3f%%)", takeProfit, 0.5)
	log.Printf("  Valore ordine: $%.2f", triggerPrice*quantity)

	longOrder, err := w.orderProcessor.PlaceLongOrder(
		w.ctx,
		symbol,
		longTriggerPrice, // trigger price
		quantity,         // quantity
		stopLoss,         // stop loss
		takeProfit,       // take profit
	)

	if err != nil {
		log.Printf("ERRORE nel piazzamento ordine LONG: %v", err)
		return ""
	}

	if !longOrder.IsSuccess() {
		log.Printf("ERRORE: Ordine rifiutato - %s (codice: %s)",
			longOrder.ErrorMessage, longOrder.ErrorCode)
		return ""
	}

	log.Printf("✅ Ordine LONG piazzato con successo!")
	log.Printf("  OrderID: %s", longOrder.OrderID)
	log.Printf("  OrderLinkID: %s", longOrder.OrderLinkID)
	log.Printf("  Status: %s", longOrder.Status)

	// ========================================
	// SALVATAGGIO NEL DATABASE
	// ========================================
	log.Println("Salvando ordine nel database...")

	// Crea l'ordine dal BybitOrderResponse
	dbOrder, err := w.createOrderFromBybitResponse(
		longOrder,
		longTriggerPrice,
		quantity,
		takeProfit,
		stopLoss,
	)
	if err != nil {
		log.Printf("❌ ERRORE: Impossibile creare ordine per database: %v", err)
		log.Println("⚠️  ATTENZIONE: Ordine piazzato su Bybit ma NON salvato nel database!")
		return longOrder.OrderID // Ritorna comunque l'ID per continuare il monitoraggio
	}

	// Salva nel database
	if err := w.saveOrderToDatabase(dbOrder); err != nil {
		log.Printf("❌ ERRORE: Impossibile salvare ordine nel database: %v", err)
		log.Println("⚠️  ATTENZIONE: Ordine piazzato su Bybit ma NON salvato nel database!")
		return longOrder.OrderID // Ritorna comunque l'ID per continuare il monitoraggio
	}

	log.Printf("✅ Ordine salvato nel database con successo!")

	// Imposta la flag orderPlaced a true
	w.orderPlaced = true
	log.Println("🔄 Flag orderPlaced impostata a true")

	return longOrder.OrderID
}

// placeShortOrder piazza un ordine SHORT
func (w *DogeTradingSystemWorker) placeShortOrder(currentPrice float64) string {
	log.Println("Placing SHORT order...")

	// Verifica che il processor sia disponibile
	if w.orderProcessor == nil {
		log.Println("ERRORE: OrderProcessor non configurato, impossibile piazzare ordine")
		return ""
	}

	// Simbolo per DOGE
	symbol := "DOGEUSDT"

	// Calcola il prezzo di trigger basato sull'ultima candela
	triggerPrice := currentPrice
	if triggerPrice <= 0 {
		log.Println("ERRORE: Impossibile calcolare il prezzo di trigger")
		return ""
	}

	// Calcola la quantità massima basata sul saldo disponibile
	quantity := w.calculateMaxQuantity(triggerPrice)
	if quantity <= 0 {
		log.Println("ERRORE: Impossibile calcolare la quantità")
		return ""
	}

	takeProfit := w.calculateShortTakeProfit(currentPrice, 0.03)
	stopLoss := w.calculateShortStopLoss(currentPrice, 0.008)

	log.Printf("Parametri ordine SHORT:")
	log.Printf("  Symbol: %s", symbol)
	log.Printf("  Trigger Price: $%.6f", triggerPrice)
	log.Printf("  Quantity: %.2f", quantity)
	log.Printf("  Stop Loss: $%.6f (%.3f%%)", stopLoss, 0.3)
	log.Printf("  Take Profit: $%.6f (%.3f%%)", takeProfit, 0.6)
	log.Printf("  Valore ordine: $%.2f", triggerPrice*quantity)

	shortTriggerPrice := currentPrice

	shortOrder, err := w.orderProcessor.PlaceShortOrder(
		w.ctx,
		symbol,
		shortTriggerPrice, // trigger price
		quantity,          // quantity
		stopLoss,          // stop loss
		takeProfit,        // take profit
	)

	if err != nil {
		log.Printf("ERRORE nel piazzamento ordine SHORT: %v", err)
		return ""
	}

	if !shortOrder.IsSuccess() {
		log.Printf("ERRORE: Ordine rifiutato - %s (codice: %s)",
			shortOrder.ErrorMessage, shortOrder.ErrorCode)
		return ""
	}

	log.Printf("✅ Ordine SHORT piazzato con successo!")
	log.Printf("  OrderID: %s", shortOrder.OrderID)
	log.Printf("  OrderLinkID: %s", shortOrder.OrderLinkID)
	log.Printf("  Status: %s", shortOrder.Status)

	// ========================================
	// SALVATAGGIO NEL DATABASE
	// ========================================
	log.Println("Salvando ordine nel database...")

	// Crea l'ordine dal BybitOrderResponse
	dbOrder, err := w.createOrderFromBybitResponse(
		shortOrder,
		shortTriggerPrice,
		quantity,
		takeProfit,
		stopLoss,
	)

	if err != nil {
		log.Printf("❌ ERRORE: Impossibile creare ordine per database: %v", err)
		log.Println("⚠️  ATTENZIONE: Ordine piazzato su Bybit ma NON salvato nel database!")
		return shortOrder.OrderID // Ritorna comunque l'ID per continuare il monitoraggio
	}

	// Salva nel database
	if err := w.saveOrderToDatabase(dbOrder); err != nil {
		log.Printf("❌ ERRORE: Impossibile salvare ordine nel database: %v", err)
		log.Println("⚠️  ATTENZIONE: Ordine piazzato su Bybit ma NON salvato nel database!")
		return shortOrder.OrderID // Ritorna comunque l'ID per continuare il monitoraggio
	}

	log.Printf("✅ Ordine salvato nel database con successo!")

	// Imposta la flag orderPlaced a true
	w.orderPlaced = true
	log.Println("🔄 Flag orderPlaced impostata a true")

	return shortOrder.OrderID
}

// ========================================
// FASE 3.1.1: Monitoraggio ordine dopo 5 minuti
// ========================================

// monitorOrDeleteOrderAfter5Minutes monitora l'ordine dopo 1 minuti
func (w *DogeTradingSystemWorker) checkPositions(orderID string) {
	log.Printf("Monitoring order %s after 1 minutes...", orderID)

	// Controlla lo stato dell'ordine
	log.Println("Checking order status...")
	orderResponse, err := w.orderProcessor.GetOrderStatus(w.ctx, "DOGEUSDT", orderID)
	if err != nil {
		log.Printf("Error getting order status: %v", err)
		return
	}

	log.Printf("Order Status: %s", orderResponse.Status)
	log.Printf("Order Details: ID=%s, Symbol=%s, Quantity=%.2f",
		orderResponse.OrderID, orderResponse.Symbol, orderResponse.Quantity)

	// Logica di gestione basata sullo stato
	switch orderResponse.Status {
	case models.OrderStatusUntriggered, models.OrderStatusNew:
		// Ordine non ancora triggerato o nuovo - cancella
		log.Println("Order not filled - cancelling...")
		cancelResponse, err := w.orderProcessor.DeleteOrder(w.ctx, "DOGEUSDT", orderID)
		if err != nil {
			log.Printf("Error cancelling order: %v", err)
			return
		}
		log.Printf("Order cancelled successfully: %s", cancelResponse.OrderID)
		// Reset della flag orderPlaced
		w.orderPlaced = false
		log.Println("🔄 Flag orderPlaced resettata a false (ordine cancellato)")

	case models.OrderStatusPartiallyFilled:
		// Ordine parzialmente fillato - cancella la parte rimanente e continua con quella fillata
		log.Println("Order partially filled - cancelling remaining quantity...")
		cancelResponse, err := w.orderProcessor.DeleteOrder(w.ctx, "DOGEUSDT", orderID)
		if err != nil {
			log.Printf("Error cancelling remaining order: %v", err)
			return
		}
		log.Printf("Remaining order cancelled successfully: %s", cancelResponse.OrderID)
		log.Println("Continuing with partially filled position...")
		// Reset della flag orderPlaced
		w.orderPlaced = false
		log.Println("🔄 Flag orderPlaced resettata a false (ordine parzialmente fillato)")

	case models.OrderStatusFilled:
		// Ordine completamente fillato - continua con il trade
		log.Println("Order fully filled - continuing with trade...")
		// Reset della flag orderPlaced
		w.orderPlaced = false
		log.Println("🔄 Flag orderPlaced resettata a false (ordine fillato)")

	case models.OrderStatusCancelled, models.OrderStatusRejected:
		// Ordine già cancellato o rifiutato - nessuna azione necessaria
		log.Printf("Order already %s - no action needed", orderResponse.Status)
		// Reset della flag orderPlaced
		w.orderPlaced = false
		log.Println("🔄 Flag orderPlaced resettata a false (ordine già cancellato/rifiutato)")

	default:
		log.Printf("Unknown order status: %s - no action taken", orderResponse.Status)
	}
}

func (w *DogeTradingSystemWorker) isPostionActive(symbol string) (bool, error) {
	positions, err := w.orderProcessor.GetPositions(w.ctx, symbol)
	if err != nil {
		log.Printf("Error getting position status: %v", err)
		return false, err
	}
	// Prima di aggioranre ho bisogtno di prendere l'id dell'ordine che ha come result "Pending"
	// RICORDA: Fai solo un ordine alla volta
	orders, err2 := w.orderService.GetOrdersByResult(w.ctx, models.OrderResultPending) // Ritorna un record, siccome al momento gestisco un solo ordine alla volta
	if err2 != nil {
		log.Printf("Error getting order: %v", err2)
		return false, err2
	}
	log.Printf("Position Status: %s", positions)
	// Se la posizione è attiva vuol dire che l'ordine è stato piazzato correttamente e chequindi devo aggioranre il DB
	if len(positions) > 0 {
		orderID := ""
		// Se l'ordine è trovato
		if len(orders) > 0 {
			orderID = orders[0].OrderID
		} else if len(orders) == 0 { // Se l'ordine non è stato trovato vuol dire che l'ordine è stato già aggioranto a "Done" in precedenza
			log.Printf("No order found with result Pending, but position is active, so the order has already been updated")
			return true, nil
		}

		// Aggiorna lo stato dell'ordine a "Done" sul DB
		err2 := w.orderService.UpdateOrderResult(w.ctx, orderID, models.OrderResultDone)
		if err2 != nil {
			log.Printf("Error updating order result: %v", err)
			return false, err
		}
	}
	return len(positions) > 0, nil
}

// ========================================
// METODI HELPER PER CALCOLI ORDINI
// ========================================

// calculateMaxQuantity calcola la quantità massima basata su prezzo e saldo disponibile
func (w *DogeTradingSystemWorker) calculateMaxQuantity(price float64) float64 {
	if price <= 0 {
		return 0
	}

	// Recupera il saldo USDT disponibile
	usdtBalance, err := w.orderProcessor.GetUSDTBalance(w.ctx)
	if err != nil {
		log.Printf("Errore nel recupero saldo USDT: %v", err)
		// Usa un valore di default se non riesce a recuperare il saldo
		return 0
	}

	availableBalance := usdtBalance
	quantity := availableBalance / price

	log.Printf("Saldo USDT disponibile: %.2f", usdtBalance)
	log.Printf("Saldo utilizzabile (90%%): %.2f", availableBalance)
	log.Printf("Quantità calcolata: %.2f", quantity)

	return quantity
}

// calculateLongStopLoss calcola il prezzo di stop loss
func (w *DogeTradingSystemWorker) calculateLongStopLoss(price, slPercentage float64) float64 {
	return price * (1 - slPercentage)
}

// calculateLongTakeProfit calcola il prezzo di take profit
func (w *DogeTradingSystemWorker) calculateLongTakeProfit(price, tpPercentage float64) float64 {
	return price * (1 + tpPercentage)
}

// calculateLongStopLoss calcola il prezzo di stop loss
func (w *DogeTradingSystemWorker) calculateShortStopLoss(price, tpPercentage float64) float64 {
	return price * (1 + tpPercentage)
}

// calculateLongTakeProfit calcola il prezzo di take profit
func (w *DogeTradingSystemWorker) calculateShortTakeProfit(price, slPercentage float64) float64 {
	return price * (1 - slPercentage)
}

func (w *DogeTradingSystemWorker) calculateGreenCandlesAverageVolume(taCandlesticks []models.Candle) float64 {
	greenCandlesAverageVolume := 0.0
	greenCandlesCount := 0

	generalCandlesCount := 0
	generalCandlesAverageVolume := 0.0

	ratio := 0.0
	for i := len(taCandlesticks) - 2; i > 0 && greenCandlesCount < 10; i-- {
		if taCandlesticks[i].Close > taCandlesticks[i].Open {
			greenCandlesAverageVolume += taCandlesticks[i].Volume
			greenCandlesCount++
		}
	}

	for i := len(taCandlesticks) - 2; i > 0 && greenCandlesCount < 10; i-- {
		greenCandlesAverageVolume += taCandlesticks[i].Volume
		generalCandlesCount++
	}
	ratio = greenCandlesAverageVolume / generalCandlesAverageVolume
	return ratio
}

func (w *DogeTradingSystemWorker) calculateRedCandlesAverageVolume(taCandlesticks []models.Candle) float64 {
	redCandlesAverageVolume := 0.0
	redCandlesCount := 0

	generalCandlesCount := 0
	generalCandlesAverageVolume := 0.0

	ratio := 0.0
	for i := len(taCandlesticks) - 2; i > 0 && redCandlesCount < 10; i-- {
		if taCandlesticks[i].Close < taCandlesticks[i].Open {
			redCandlesAverageVolume += taCandlesticks[i].Volume
			redCandlesCount++
		}
	}

	for i := len(taCandlesticks) - 2; i > 0 && generalCandlesCount < 10; i-- {
		generalCandlesAverageVolume += taCandlesticks[i].Volume
		generalCandlesCount++
	}
	ratio = redCandlesAverageVolume / generalCandlesAverageVolume
	return ratio
}

func (w *DogeTradingSystemWorker) calculateGreenCandlesVolumeMax(taCandlesticks []models.Candle) float64 {
	greenCandlesVolumeMax := 0.0
	greenCandlesCount := 0
	for i := len(taCandlesticks) - 2; i > 0 && greenCandlesCount < 10; i-- {
		if taCandlesticks[i].Close > taCandlesticks[i].Open {
			if greenCandlesVolumeMax < taCandlesticks[i].Volume {
				greenCandlesVolumeMax = taCandlesticks[i].Volume
			}
			greenCandlesCount++
		}
	}
	return greenCandlesVolumeMax
}

func (w *DogeTradingSystemWorker) calculateRedCandlesVolumeMax(taCandlesticks []models.Candle) float64 {
	redCandlesVolumeMax := 0.0
	redCandlesCount := 0
	for i := len(taCandlesticks) - 2; i > 0 && redCandlesCount < 10; i-- {
		if taCandlesticks[i].Close < taCandlesticks[i].Open {
			if redCandlesVolumeMax < taCandlesticks[i].Volume {
				redCandlesVolumeMax = taCandlesticks[i].Volume
			}
			redCandlesCount++
		}
	}
	return redCandlesVolumeMax
}
