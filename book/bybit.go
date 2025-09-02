package book

import (
	"context"
	"cross-exchange-arbitrage/models"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// BybitOrderBookStreamer implementa l'interfaccia OrderBookStreamer per Bybit
type BybitOrderBookStreamer struct {
	wsURL string
	conn  *websocket.Conn
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

// NewBybitOrderBookStreamer crea una nuova istanza di BybitOrderBookStreamer
func NewBybitOrderBookStreamer() *BybitOrderBookStreamer {
	return &BybitOrderBookStreamer{
		wsURL: "wss://stream.bybit.com/v5/public/spot",
	}
}

// OrderBookStream implementa il metodo dell'interfaccia OrderBookStreamer
func (b *BybitOrderBookStreamer) OrderBookStream(
	ctx context.Context,
	symbol string,
	depth int,
	updateChan chan<- *models.OrderBookData,
	errChan chan<- error,
) error {
	// Verifica che la depth sia valida (Bybit supporta 1/50/200)
	if depth > 50 {
		depth = 50 // Limitiamo a 50 per performance
	}

	// Connessione WebSocket
	var err error
	b.conn, _, err = websocket.DefaultDialer.DialContext(ctx, b.wsURL, nil)
	if err != nil {
		return fmt.Errorf("errore connessione WebSocket Bybit: %w", err)
	}

	// Cleanup alla chiusura
	defer func() {
		if b.conn != nil {
			b.conn.Close()
		}
	}()

	// Sottoscrizione al topic dell'orderbook
	subscribeMsg := BybitSubscriptionMessage{
		Op:   "subscribe",
		Args: []string{fmt.Sprintf("orderbook.%d.%s", depth, symbol)},
	}

	if err := b.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("errore sottoscrizione simbolo %s: %w", symbol, err)
	}

	// Goroutine per la gestione dei messaggi
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, message, err := b.conn.ReadMessage()
				if err != nil {
					errChan <- fmt.Errorf("errore lettura messaggio WebSocket: %w", err)
					return
				}

				var response BybitOrderBookResponse
				if err := json.Unmarshal(message, &response); err != nil {
					// Ignora messaggi che non sono order book updates
					continue
				}

				// Processa solo messaggi dell'order book
				if response.Topic != "" && response.Data.Symbol != "" {
					orderBookData, err := b.convertToOrderBookData(&response)
					if err != nil {
						errChan <- fmt.Errorf("errore conversione dati orderbook: %w", err)
						continue
					}

					// Invia l'aggiornamento sul canale
					select {
					case updateChan <- orderBookData:
					default:
						// Se il canale è pieno, logga e continua
						log.Printf("Canale orderbook pieno, skip update per %s", symbol)
					}
				}
			}
		}
	}()

	return nil
}

// convertToOrderBookData converte la risposta di Bybit nel formato OrderBookData
func (b *BybitOrderBookStreamer) convertToOrderBookData(response *BybitOrderBookResponse) (*models.OrderBookData, error) {
	if len(response.Data.Bids) == 0 || len(response.Data.Asks) == 0 {
		return nil, fmt.Errorf("orderbook vuoto")
	}

	// Converti i bid
	bids := make([]models.OrderBookLevel, 0, len(response.Data.Bids))
	for _, bid := range response.Data.Bids {
		if len(bid) != 2 {
			continue
		}
		price, err := strconv.ParseFloat(bid[0], 64)
		if err != nil {
			continue
		}
		quantity, err := strconv.ParseFloat(bid[1], 64)
		if err != nil {
			continue
		}
		bids = append(bids, models.OrderBookLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Converti gli ask
	asks := make([]models.OrderBookLevel, 0, len(response.Data.Asks))
	for _, ask := range response.Data.Asks {
		if len(ask) != 2 {
			continue
		}
		price, err := strconv.ParseFloat(ask[0], 64)
		if err != nil {
			continue
		}
		quantity, err := strconv.ParseFloat(ask[1], 64)
		if err != nil {
			continue
		}
		asks = append(asks, models.OrderBookLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Crea l'OrderBookData
	orderBookData := &models.OrderBookData{
		Symbol:    response.Data.Symbol,
		BestBid:   bids[0],
		BestAsk:   asks[0],
		Bids:      bids,
		Asks:      asks,
		Exchange:  "bybit",
		Timestamp: time.Unix(response.Ts/1000, (response.Ts%1000)*1000000),
	}

	return orderBookData, nil
}

