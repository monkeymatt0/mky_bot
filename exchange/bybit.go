package exchange

import (
	"context"
	"cross-exchange-arbitrage/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// URL di base per le API REST di Bybit
	// bybitRESTBaseURL = "https://api.bybit.com"
	bybitRESTBaseURL = "https://api-testnet.bybit.com"

	// Endpoint per le candele
	bybitKlineEndpoint = "/v5/market/kline"

	// Limite massimo di candele per richiesta
	maxCandlesPerRequest = 1000

	// Intervallo tra le richieste per evitare rate limiting
	requestInterval = time.Second
)

// BybitExchange implementa l'interfaccia Exchange per Bybit
type BybitExchange struct {
	wsURL      string
	conn       *websocket.Conn
	priceData  map[string]*models.RealTimePriceData
	subscriber map[string]chan *models.RealTimePriceData
	httpClient *http.Client
	testnet    bool
}

// BybitOrderBookResponse rappresenta la risposta dell'order book di Bybit
type BybitOrderBookResponse struct {
	Topic string `json:"topic"`
	Type  string `json:"type"`
	Data  struct {
		Symbol   string     `json:"s"`
		Bids     [][]string `json:"b"`
		Asks     [][]string `json:"a"`
		UpdateID int64      `json:"u"`
		Seq      int64      `json:"seq"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

// BybitSubscriptionMessage rappresenta il messaggio di sottoscrizione
type BybitSubscriptionMessage struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// NewBybitExchange crea una nuova istanza di BybitExchange
// BybitKlineResponse rappresenta la risposta delle candele di Bybit
type BybitKlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string     `json:"category"`
		Symbol   string     `json:"symbol"`
		List     [][]string `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

func NewBybitExchange(testnet bool) *BybitExchange {
	if testnet {
		return &BybitExchange{
			wsURL:      "wss://stream.bybit.com/v5/public/linear",
			priceData:  make(map[string]*models.RealTimePriceData),
			subscriber: make(map[string]chan *models.RealTimePriceData),
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			testnet: true,
		}
	}

	return &BybitExchange{
		wsURL:      "wss://stream.bybit.com/v5/public/linear",
		priceData:  make(map[string]*models.RealTimePriceData),
		subscriber: make(map[string]chan *models.RealTimePriceData),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect stabilisce la connessione WebSocket con Bybit
func (b *BybitExchange) Connect(ctx context.Context) error {
	var err error
	b.conn, _, err = websocket.DefaultDialer.DialContext(ctx, b.wsURL, nil)
	if err != nil {
		return fmt.Errorf("errore connessione WebSocket Bybit: %w", err)
	}

	// Avvia il listener per i messaggi WebSocket
	go b.messageListener(ctx)

	log.Println("Connessione WebSocket Bybit stabilita")
	return nil
}

// Subscribe sottoscrive agli aggiornamenti dell'order book per un simbolo
func (b *BybitExchange) Subscribe(symbol string) error {
	if b.conn == nil {
		return fmt.Errorf("connessione WebSocket non stabilita")
	}

	subscribeMsg := BybitSubscriptionMessage{
		Op:   "subscribe",
		Args: []string{fmt.Sprintf("orderbook.1.%s", symbol)},
	}

	if err := b.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("errore sottoscrizione simbolo %s: %w", symbol, err)
	}

	log.Printf("Sottoscritto agli aggiornamenti dell'order book per %s", symbol)
	return nil
}

// messageListener ascolta i messaggi WebSocket e aggiorna i dati dei prezzi
func (b *BybitExchange) messageListener(ctx context.Context) {
	defer func() {
		if b.conn != nil {
			b.conn.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := b.conn.ReadMessage()
			if err != nil {
				log.Printf("Errore lettura messaggio WebSocket: %v", err)
				return
			}

			var response BybitOrderBookResponse
			if err := json.Unmarshal(message, &response); err != nil {
				// Ignora messaggi che non sono order book updates
				continue
			}

			// Processa solo messaggi dell'order book
			if response.Topic != "" && response.Data.Symbol != "" {
				b.processOrderBookUpdate(&response)
			}
		}
	}
}

// processOrderBookUpdate processa gli aggiornamenti dell'order book
func (b *BybitExchange) processOrderBookUpdate(response *BybitOrderBookResponse) {
	symbol := response.Data.Symbol

	// Verifica che abbiamo almeno un bid e un ask
	if len(response.Data.Bids) == 0 || len(response.Data.Asks) == 0 {
		return
	}

	// Estrae il miglior bid (primo elemento)
	bestBidPrice, err := strconv.ParseFloat(response.Data.Bids[0][0], 64)
	if err != nil {
		log.Printf("Errore parsing bid price: %v", err)
		return
	}
	bestBidQty, err := strconv.ParseFloat(response.Data.Bids[0][1], 64)
	if err != nil {
		log.Printf("Errore parsing bid quantity: %v", err)
		return
	}

	// Estrae il miglior ask (primo elemento)
	bestAskPrice, err := strconv.ParseFloat(response.Data.Asks[0][0], 64)
	if err != nil {
		log.Printf("Errore parsing ask price: %v", err)
		return
	}
	bestAskQty, err := strconv.ParseFloat(response.Data.Asks[0][1], 64)
	if err != nil {
		log.Printf("Errore parsing ask quantity: %v", err)
		return
	}

	// Calcola il prezzo medio
	midPrice := (bestBidPrice + bestAskPrice) / 2

	// Crea i dati del prezzo in tempo reale
	priceData := &models.RealTimePriceData{
		Symbol:       symbol,
		Price:        midPrice,
		BidPrice:     bestBidPrice,
		AskPrice:     bestAskPrice,
		BidLiquidity: bestBidQty,
		AskLiquidity: bestAskQty,
		Exchange:     "bybit",
		Timestamp:    time.Unix(response.Ts/1000, (response.Ts%1000)*1000000),
	}

	// Aggiorna i dati interni
	b.priceData[symbol] = priceData

	// Notifica i subscriber se presenti
	if ch, exists := b.subscriber[symbol]; exists {
		select {
		case ch <- priceData:
		default:
			// Channel pieno, salta questo aggiornamento
		}
	}

	// Log dell'aggiornamento
	log.Printf("PREZZO: %.4f, BID: %.4f (LIQUIDITA: %.4f), ASK: %.4f (LIQUIDITA: %.4f) - %s",
		priceData.Price, priceData.BidPrice, priceData.BidLiquidity,
		priceData.AskPrice, priceData.AskLiquidity, symbol)
}

// GetRealTimePrice implementa l'interfaccia Exchange
func (b *BybitExchange) GetRealTimePrice(ctx context.Context, symbol string) (*models.RealTimePriceData, error) {
	// Se non siamo connessi, stabilisci la connessione
	if b.conn == nil {
		if err := b.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// Se non siamo già sottoscritti a questo simbolo, sottoscriviti
	if _, exists := b.priceData[symbol]; !exists {
		if err := b.Subscribe(symbol); err != nil {
			return nil, err
		}

		// Crea un channel per questo simbolo se non esiste
		if _, exists := b.subscriber[symbol]; !exists {
			b.subscriber[symbol] = make(chan *models.RealTimePriceData, 10)
		}
	}

	// Aspetta il primo aggiornamento o usa quello cached
	if priceData, exists := b.priceData[symbol]; exists {
		return priceData, nil
	}

	// Aspetta il primo aggiornamento
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case priceData := <-b.subscriber[symbol]:
		return priceData, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout: nessun dato ricevuto per %s entro 10 secondi", symbol)
	}
}

// Close chiude la connessione WebSocket
func (b *BybitExchange) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

// GetLatestPrice restituisce l'ultimo prezzo cached per un simbolo
func (b *BybitExchange) GetLatestPrice(symbol string) (*models.RealTimePriceData, bool) {
	priceData, exists := b.priceData[symbol]
	return priceData, exists
}

// SubscribeToUpdates restituisce un channel per ricevere aggiornamenti in tempo reale
func (b *BybitExchange) SubscribeToUpdates(symbol string) <-chan *models.RealTimePriceData {
	if _, exists := b.subscriber[symbol]; !exists {
		b.subscriber[symbol] = make(chan *models.RealTimePriceData, 10)
	}
	return b.subscriber[symbol]
}

// FetchLastCandles implementa l'interfaccia Exchange
func (b *BybitExchange) FetchLastCandles(ctx context.Context, symbol string, market models.Market, timeframe models.Timeframe, limit int) (*models.CandleResponse, error) {
	// Se il market non è specificato, usa derivatives di default
	if market == "" {
		market = models.DerivativesMarket
	}

	// Converti il market nel formato di Bybit
	category := "linear"
	if market == models.SpotMarket {
		category = "spot"
	}

	// Inizializza la risposta
	response := &models.CandleResponse{
		Candles: make([]models.Candle, 0, limit),
		HasMore: false,
	}

	// Calcola quante richieste sono necessarie
	remainingCandles := limit
	var startTime *int64 // timestamp per la paginazione

	for remainingCandles > 0 {
		// Controlla se il contesto è stato cancellato
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calcola il limite per questa richiesta
		requestLimit := remainingCandles
		if requestLimit > maxCandlesPerRequest {
			requestLimit = maxCandlesPerRequest
		}

		// Costruisci l'URL
		url := fmt.Sprintf("%s%s?category=%s&symbol=%s&interval=%s&limit=%d",
			bybitRESTBaseURL, bybitKlineEndpoint, category, symbol, timeframe, requestLimit)

		// Aggiungi il timestamp di inizio se presente
		if startTime != nil {
			url = fmt.Sprintf("%s&start=%d", url, *startTime)
		}

		// Esegui la richiesta
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("errore creazione richiesta: %w", err)
		}

		resp, err := b.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("errore esecuzione richiesta: %w", err)
		}

		// Leggi il corpo della risposta
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("errore lettura risposta: %w", err)
		}

		// Decodifica la risposta
		var klineResp BybitKlineResponse
		if err := json.Unmarshal(body, &klineResp); err != nil {
			return nil, fmt.Errorf("errore decodifica risposta: %w", err)
		}

		// Verifica se ci sono errori
		if klineResp.RetCode != 0 {
			return nil, fmt.Errorf("errore API Bybit: %s", klineResp.RetMsg)
		}

		// Processa le candele
		// Bybit restituisce le candele in ordine decrescente (più recenti prima)
		// Formato: [timestamp, open, high, low, close, volume, turnover]
		for _, data := range klineResp.Result.List {
			if len(data) < 6 {
				continue
			}

			// Converti i valori
			timestamp, _ := strconv.ParseInt(data[0], 10, 64)
			open, _ := strconv.ParseFloat(data[1], 64)
			high, _ := strconv.ParseFloat(data[2], 64)
			low, _ := strconv.ParseFloat(data[3], 64)
			close, _ := strconv.ParseFloat(data[4], 64)
			volume, _ := strconv.ParseFloat(data[5], 64)

			candle := models.Candle{
				Timestamp: time.UnixMilli(timestamp),
				Open:      open,
				High:      high,
				Low:       low,
				Close:     close,
				Volume:    volume,
			}

			response.Candles = append(response.Candles, candle)
		}

		// Aggiorna il conteggio delle candele rimanenti
		remainingCandles -= len(klineResp.Result.List)

		// Se non abbiamo ricevuto il numero massimo di candele, non ci sono più dati
		if len(klineResp.Result.List) < requestLimit {
			break
		}

		// Aggiorna il timestamp di inizio per la prossima richiesta
		if len(klineResp.Result.List) > 0 {
			ts, _ := strconv.ParseInt(klineResp.Result.List[len(klineResp.Result.List)-1][0], 10, 64)
			startTime = &ts
		}

		// Aspetta un secondo prima della prossima richiesta
		time.Sleep(requestInterval)
	}

	// Indica se ci sono altre candele disponibili
	response.HasMore = remainingCandles > 0

	return response, nil
}
