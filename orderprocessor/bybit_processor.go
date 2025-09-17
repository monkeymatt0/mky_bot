package orderprocessor

import (
	"bytes"
	"context"
	"cross-exchange-arbitrage/models"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	// URL di base per le API REST di Bybit
	bybitAPIBaseURL = "https://api.bybit.com"

	// Endpoint per piazzare ordini
	bybitPlaceOrderEndpoint = "/v5/order/create"

	// Endpoint per cancellare ordini
	bybitCancelOrderEndpoint = "/v5/order/cancel"

	// Endpoint per aggiornare stop loss e take profit
	bybitUpdateTradingStopEndpoint = "/v5/position/trading-stop"

	// Endpoint per ottenere stato ordini in tempo reale
	bybitGetOrderStatusEndpoint = "/v5/order/realtime"

	// Endpoint per ottenere le posizioni attive
	bybitGetPositionsEndpoint = "/v5/position/list"

	// Endpoint per ottenere il saldo del wallet
	bybitGetWalletBalanceEndpoint = "/v5/account/wallet-balance"

	// Categoria per mercati derivati perpetual
	derivativesCategory = "linear"
)

// RICORDA:
/*
Questo processore è fatto per DOGEUSDT che non accetta numeri decimali per la quantità.
Quindi se nel caso dovresti usare questo processore per un altro simbolo dovrai implementare le opportune modifiche.
*/

// BybitOrderProcessor implementa OrderProcessor per Bybit
type BybitOrderProcessor struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewBybitOrderProcessor crea una nuova istanza di BybitOrderProcessor
func NewBybitOrderProcessor(apiKey, apiSecret string) *BybitOrderProcessor {
	return &BybitOrderProcessor{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BybitAPIResponse rappresenta la risposta standard delle API Bybit
type BybitAPIResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID     string  `json:"orderId"`
		OrderLinkID string  `json:"orderLinkId"`
		AvgPrice    float64 `json:"avgPrice"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// BybitCancelOrderRequest rappresenta la richiesta di cancellazione ordine
type BybitCancelOrderRequest struct {
	Category    string `json:"category"`              // "linear" per derivatives
	Symbol      string `json:"symbol"`                // Es. "BTCUSDT"
	OrderID     string `json:"orderId,omitempty"`     // ID ordine (opzionale se si usa orderLinkId)
	OrderLinkID string `json:"orderLinkId,omitempty"` // ID cliente (opzionale se si usa orderId)
}

// BybitCancelOrderResponse rappresenta la risposta di cancellazione ordine
type BybitCancelOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// BybitUpdateTradingStopRequest rappresenta la richiesta di aggiornamento trading stop
type BybitUpdateTradingStopRequest struct {
	Category     string  `json:"category"`               // "linear" per derivatives
	Symbol       string  `json:"symbol"`                 // Es. "DOGEUSDT"
	TpslMode     string  `json:"tpslMode"`               // "Full" per posizione intera, "Partial" per parziale
	PositionIdx  int     `json:"positionIdx"`            // Indice posizione (0: one-way mode)
	TakeProfit   string  `json:"takeProfit,omitempty"`   // Prezzo take profit come stringa
	StopLoss     string  `json:"stopLoss,omitempty"`     // Prezzo stop loss come stringa
	TrailingStop *string `json:"trailingStop,omitempty"` // Trailing stop per distanza prezzo
	TpTriggerBy  string  `json:"tpTriggerBy,omitempty"`  // Trigger per TP: "LastPrice", "IndexPrice", "MarkPrice"
	SlTriggerBy  string  `json:"slTriggerBy,omitempty"`  // Trigger per SL: "LastPrice", "IndexPrice", "MarkPrice"
	ActivePrice  *string `json:"activePrice,omitempty"`  // Prezzo trigger per trailing stop
	TpSize       *string `json:"tpSize,omitempty"`       // Dimensione take profit (per modalità Partial)
	SlSize       *string `json:"slSize,omitempty"`       // Dimensione stop loss (per modalità Partial)
	TpLimitPrice *string `json:"tpLimitPrice,omitempty"` // Prezzo limite per TP (per ordini Limit)
	SlLimitPrice *string `json:"slLimitPrice,omitempty"` // Prezzo limite per SL (per ordini Limit)
	TpOrderType  string  `json:"tpOrderType,omitempty"`  // Tipo ordine TP: "Market", "Limit"
	SlOrderType  string  `json:"slOrderType,omitempty"`  // Tipo ordine SL: "Market", "Limit"
}

// BybitUpdateTradingStopResponse rappresenta la risposta di aggiornamento trading stop
type BybitUpdateTradingStopResponse struct {
	RetCode int      `json:"retCode"`
	RetMsg  string   `json:"retMsg"`
	Result  struct{} `json:"result"` // Bybit ritorna un oggetto vuoto per questa API
	Time    int64    `json:"time"`
}

// BybitOrderStatusResponse rappresenta la risposta con lo stato dell'ordine
type BybitOrderStatusResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
			Symbol      string `json:"symbol"`
			OrderStatus string `json:"orderStatus"` // New, PartiallyFilled, Untriggered, Rejected, PartiallyFilledCanceled, Filled, Deactivated, Triggered, Cancelled
			Side        string `json:"side"`
			OrderType   string `json:"orderType"`
			Price       string `json:"price"`
			Qty         string `json:"qty"`
			CreatedTime string `json:"createdTime"`
			UpdatedTime string `json:"updatedTime"`
		} `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// PlaceLongOrder implementa l'interfaccia OrderProcessor per ordini long
// Usa ordini Market per esecuzione immediata
func (bp *BybitOrderProcessor) PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	// Genera un ID univoco per l'ordine
	orderLinkID := fmt.Sprintf("long_%s_%d", symbol, time.Now().Unix())

	//{
	//   "symbol": "BTCUSDT",
	//   "side": "Buy",
	//   "orderType": "Market",
	//   "qty": "0.01",
	//   "timeInForce": "IOC",
	//   "reduceOnly": false
	//}

	// Crea la richiesta di ordine Market per LONG (esecuzione immediata)
	orderReq := models.OrderRequest{
		Category:    derivativesCategory,
		Symbol:      symbol,
		Side:        models.OrderSideBuy,
		OrderType:   models.OrderTypeMarket,
		Qty:         strconv.FormatFloat(math.Floor(quantity), 'f', 0, 64),
		TimeInForce: models.TimeInForceIOC,
		OrderLinkId: orderLinkID,
		ReduceOnly:  false,
		StopLoss:    strconv.FormatFloat(stopLoss, 'f', 2, 64),
		TakeProfit:  strconv.FormatFloat(takeProfit, 'f', 2, 64),
	}

	return bp.placeOrder(ctx, &orderReq, takeProfit, stopLoss)
}

// PlaceShortOrder implementa l'interfaccia OrderProcessor per ordini short
// Usa ordini Stop per vendere quando il prezzo raggiunge il livello specificato
func (bp *BybitOrderProcessor) PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	// Genera un ID univoco per l'ordine
	orderLinkID := fmt.Sprintf("short_%s_%d", symbol, time.Now().Unix())

	// Crea la richiesta di ordine Market per SHORT (esecuzione immediata)
	orderReq := models.OrderRequest{
		Category:    derivativesCategory,
		Symbol:      symbol,
		Side:        models.OrderSideSell,
		OrderType:   models.OrderTypeMarket,
		Qty:         strconv.FormatFloat(math.Floor(quantity), 'f', 0, 64),
		TimeInForce: models.TimeInForceIOC,
		OrderLinkId: orderLinkID,
		ReduceOnly:  false,
		StopLoss:    strconv.FormatFloat(stopLoss, 'f', 2, 64),
		TakeProfit:  strconv.FormatFloat(takeProfit, 'f', 2, 64),
	}

	return bp.placeOrder(ctx, &orderReq, takeProfit, stopLoss)
}

// placeOrder invia l'ordine a Bybit usando le API autenticate
func (bp *BybitOrderProcessor) placeOrder(ctx context.Context, orderReq *models.OrderRequest, takeProfit, stopLoss float64) (*models.OrderResponse, error) {

	// Serializza la richiesta in JSON
	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		return nil, fmt.Errorf("errore nella serializzazione dell'ordine: %w", err)
	}

	// Crea la richiesta HTTP
	url := bybitAPIBaseURL + bybitPlaceOrderEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers necessari per l'autenticazione Bybit
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"

	// Calcola la firma HMAC
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var apiResp BybitAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Converte la risposta nel formato interno
	orderResp := &models.OrderResponse{
		OrderID:      apiResp.Result.OrderID,
		OrderLinkID:  apiResp.Result.OrderLinkID,
		AveragePrice: apiResp.Result.AvgPrice,
		Symbol:       orderReq.Symbol,
		Side:         orderReq.Side,
		OrderType:    orderReq.OrderType,
		CreatedTime:  time.Unix(apiResp.Time/1000, 0),
		UpdatedTime:  time.Unix(apiResp.Time/1000, 0),
		ErrorCode:    strconv.Itoa(apiResp.RetCode),
		ErrorMessage: apiResp.RetMsg,
	}

	// Converte i valori string in float64
	if orderReq.Price != "" {
		orderResp.Price, _ = strconv.ParseFloat(orderReq.Price, 64)
	}
	if orderReq.TriggerPrice != "" {
		orderResp.TriggerPrice, _ = strconv.ParseFloat(orderReq.TriggerPrice, 64)
	}
	if orderReq.Qty != "" {
		orderResp.Quantity, _ = strconv.ParseFloat(orderReq.Qty, 64)
	}
	if orderReq.StopLoss != "" {
		orderResp.StopLoss, _ = strconv.ParseFloat(orderReq.StopLoss, 64)
	}
	if orderReq.TakeProfit != "" {
		orderResp.TakeProfit, _ = strconv.ParseFloat(orderReq.TakeProfit, 64)
	}

	// Imposta lo status iniziale
	if apiResp.RetCode == 0 {
		orderResp.Status = models.OrderStatusUntriggered // Ordine stop non ancora triggerato
	} else {
		orderResp.Status = models.OrderStatusRejected
	}

	return orderResp, nil
}

// DeleteOrder cancella un ordine esistente usando l'orderID o orderLinkID
// Accetta sia l'ID dell'ordine di Bybit che l'ID cliente personalizzato
func (bp *BybitOrderProcessor) DeleteOrder(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error) {
	// Crea la richiesta di cancellazione
	cancelReq := BybitCancelOrderRequest{
		Category: derivativesCategory,
		Symbol:   symbol,
	}

	// Determina se è un orderID (UUID format) o orderLinkID (nostro formato personalizzato)
	if isUUIDFormat(orderID) {
		cancelReq.OrderID = orderID
	} else {
		cancelReq.OrderLinkID = orderID
	}

	// Serializza la richiesta in JSON
	jsonData, err := json.Marshal(cancelReq)
	if err != nil {
		return nil, fmt.Errorf("errore nella serializzazione della cancellazione: %w", err)
	}

	// Crea la richiesta HTTP
	url := bybitAPIBaseURL + bybitCancelOrderEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta di cancellazione: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var cancelResp BybitCancelOrderResponse
	if err := json.Unmarshal(body, &cancelResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Converte la risposta nel formato interno
	orderResp := &models.OrderResponse{
		OrderID:      cancelResp.Result.OrderID,
		OrderLinkID:  cancelResp.Result.OrderLinkID,
		Symbol:       symbol,
		Status:       models.OrderStatusCancelled,
		CreatedTime:  time.Unix(cancelResp.Time/1000, 0),
		UpdatedTime:  time.Unix(cancelResp.Time/1000, 0),
		ErrorCode:    strconv.Itoa(cancelResp.RetCode),
		ErrorMessage: cancelResp.RetMsg,
	}

	// Determina lo status finale
	if cancelResp.RetCode == 0 {
		orderResp.Status = models.OrderStatusCancelled
	} else {
		orderResp.Status = models.OrderStatusRejected
	}

	return orderResp, nil
}

// setTradingStop imposta stop loss e take profit per una posizione
// Metodo interno per gestire il posizionamento di TP/SL dopo un ordine
func (bp *BybitOrderProcessor) setTradingStop(ctx context.Context, symbol string, side models.OrderSide, takeProfit, stopLoss float64) error {
	// Prima verifica che la posizione esista
	positions, err := bp.GetPositions(ctx, symbol)
	if err != nil {
		return fmt.Errorf("errore nel recupero posizioni: %w", err)
	}

	if len(positions) == 0 {
		// Aspetta un secondo
		time.Sleep(1 * time.Second)
		positions, err = bp.GetPositions(ctx, symbol)
		if err != nil {
			return fmt.Errorf("errore nel recupero posizioni: %w", err)
		}
		if len(positions) == 0 {
			return fmt.Errorf("nessuna posizione trovata per il simbolo %s", symbol)
		}
	}

	// Crea la richiesta per il trading stop
	tradingStopReq := BybitUpdateTradingStopRequest{
		Category:    derivativesCategory,
		Symbol:      symbol,
		TpslMode:    "Full", // tutta la posizione
		PositionIdx: 0,      // one-way mode
		TpTriggerBy: "LastPrice",
		SlTriggerBy: "LastPrice",
	}

	// Aggiungi solo i valori > 0 per evitare conflitti
	if takeProfit > 0 {
		tradingStopReq.TakeProfit = strconv.FormatFloat(takeProfit, 'f', 2, 64)
	}

	if stopLoss > 0 {
		tradingStopReq.StopLoss = strconv.FormatFloat(stopLoss, 'f', 2, 64)
	}

	// Se non c'è nulla da aggiornare, esci
	if tradingStopReq.TakeProfit == "" && tradingStopReq.StopLoss == "" {
		return fmt.Errorf("nessun valore TP/SL da impostare")
	}

	// Log della richiesta per debug
	fmt.Printf("Impostando trading stop per %s - TP: %v, SL: %v\n", symbol, tradingStopReq.TakeProfit, tradingStopReq.StopLoss)

	// Serializza la richiesta in JSON
	jsonData, err := json.Marshal(tradingStopReq)
	if err != nil {
		return fmt.Errorf("errore nella serializzazione della richiesta trading stop: %w", err)
	}

	// Crea la richiesta HTTP
	url := bybitAPIBaseURL + bybitUpdateTradingStopEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("errore nell'esecuzione della richiesta trading stop: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("errore nella lettura della risposta trading stop: %w", err)
	}

	// Decodifica la risposta
	var tradingStopResp BybitUpdateTradingStopResponse
	if err := json.Unmarshal(body, &tradingStopResp); err != nil {
		return fmt.Errorf("errore nella decodifica della risposta trading stop: %w", err)
	}

	// Verifica che la richiesta sia andata a buon fine
	if tradingStopResp.RetCode != 0 {
		// Gestione specifica per errore 34040 (not modified)
		if tradingStopResp.RetCode == 34040 {
			return fmt.Errorf("trading stop non modificato: i valori potrebbero essere identici a quelli esistenti o la posizione non è pronta (codice: %d)", tradingStopResp.RetCode)
		}
		return fmt.Errorf("errore API Bybit nel trading stop: %s (codice: %d)", tradingStopResp.RetMsg, tradingStopResp.RetCode)
	}

	fmt.Printf("Trading stop impostato con successo per %s\n", symbol)
	return nil
}

// UpdateOrder aggiorna stop loss e/o take profit di una posizione esistente
// Accetta parametri flessibili - può aggiornare solo SL, solo TP, o entrambi
func (bp *BybitOrderProcessor) UpdateOrder(ctx context.Context, params UpdateOrderParams) (*models.OrderResponse, error) {
	// Valida che almeno uno tra StopLoss e TakeProfit sia specificato
	if params.StopLoss == nil && params.TakeProfit == nil {
		return nil, fmt.Errorf("almeno uno tra StopLoss e TakeProfit deve essere specificato")
	}

	// Crea la richiesta di aggiornamento
	updateReq := BybitUpdateTradingStopRequest{
		Category:    derivativesCategory,
		Symbol:      params.Symbol,
		PositionIdx: params.PositionIdx,
		TpTriggerBy: "LastPrice", // Usa sempre LastPrice come default
		SlTriggerBy: "LastPrice", // Usa sempre LastPrice come default
	}

	// Converte StopLoss in stringa se specificato
	if params.StopLoss != nil {
		updateReq.StopLoss = strconv.FormatFloat(*params.StopLoss, 'f', 2, 64)
	}

	// Converte TakeProfit in stringa se specificato
	if params.TakeProfit != nil {
		updateReq.TakeProfit = strconv.FormatFloat(*params.TakeProfit, 'f', 2, 64)
	}

	// Serializza la richiesta in JSON
	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return nil, fmt.Errorf("errore nella serializzazione della richiesta di aggiornamento: %w", err)
	}

	// Crea la richiesta HTTP
	url := bybitAPIBaseURL + bybitUpdateTradingStopEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta di aggiornamento: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var updateResp BybitUpdateTradingStopResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Converte la risposta nel formato interno
	orderResp := &models.OrderResponse{
		Symbol:       params.Symbol,
		CreatedTime:  time.Unix(updateResp.Time/1000, 0),
		UpdatedTime:  time.Unix(updateResp.Time/1000, 0),
		ErrorCode:    strconv.Itoa(updateResp.RetCode),
		ErrorMessage: updateResp.RetMsg,
	}

	// Aggiorna i valori modificati
	if params.StopLoss != nil {
		orderResp.StopLoss = *params.StopLoss
	}
	if params.TakeProfit != nil {
		orderResp.TakeProfit = *params.TakeProfit
	}

	// Determina lo status finale
	if updateResp.RetCode == 0 {
		orderResp.Status = models.OrderStatusNew // Posizione aggiornata con successo
	} else {
		orderResp.Status = models.OrderStatusRejected
	}

	return orderResp, nil
}

// GetOrderStatus recupera lo stato di un ordine specifico
// Accetta sia orderID (UUID di Bybit) che orderLinkID (ID cliente personalizzato)
func (bp *BybitOrderProcessor) GetOrderStatus(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error) {
	// Costruisce l'URL con parametri query
	baseURL := bybitAPIBaseURL + bybitGetOrderStatusEndpoint

	// Crea i parametri della query
	params := url.Values{}
	params.Set("category", derivativesCategory)
	params.Set("symbol", symbol)

	// Determina se è un orderID (UUID format) o orderLinkID (nostro formato personalizzato)
	if isUUIDFormat(orderID) {
		params.Set("orderId", orderID)
	} else {
		params.Set("orderLinkId", orderID)
	}

	// URL completo con parametri
	fullURL := baseURL + "?" + params.Encode()

	// Crea la richiesta HTTP GET
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"

	// Per richieste GET, il payload per la firma è costituito dai parametri query
	queryString := params.Encode()
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, queryString)

	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var statusResp BybitOrderStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Verifica che la richiesta sia andata a buon fine
	if statusResp.RetCode != 0 {
		return nil, fmt.Errorf("errore API Bybit: %s (codice: %d)", statusResp.RetMsg, statusResp.RetCode)
	}

	// Verifica che sia stato trovato almeno un ordine
	if len(statusResp.Result.List) == 0 {
		return nil, fmt.Errorf("ordine non trovato: %s", orderID)
	}

	// Prende il primo ordine dalla lista (dovrebbe essere l'unico)
	order := statusResp.Result.List[0]

	// Converte la risposta nel formato interno
	orderResp := &models.OrderResponse{
		OrderID:      order.OrderID,
		OrderLinkID:  order.OrderLinkID,
		Symbol:       order.Symbol,
		Side:         models.OrderSide(order.Side),
		OrderType:    models.OrderType(order.OrderType),
		Status:       models.OrderStatus(order.OrderStatus),
		ErrorCode:    strconv.Itoa(statusResp.RetCode),
		ErrorMessage: statusResp.RetMsg,
	}

	// Converte i valori string in float64
	if order.Price != "" {
		orderResp.Price, _ = strconv.ParseFloat(order.Price, 64)
	}
	if order.Qty != "" {
		orderResp.Quantity, _ = strconv.ParseFloat(order.Qty, 64)
	}

	// Converte i timestamp
	if createdTimeInt, err := strconv.ParseInt(order.CreatedTime, 10, 64); err == nil {
		orderResp.CreatedTime = time.Unix(createdTimeInt/1000, 0)
	}
	if updatedTimeInt, err := strconv.ParseInt(order.UpdatedTime, 10, 64); err == nil {
		orderResp.UpdatedTime = time.Unix(updatedTimeInt/1000, 0)
	}

	return orderResp, nil
}

// GetPositions recupera le posizioni attive per un simbolo specifico
// Se symbol è vuoto, usa "USDT" come settleCoin per ottenere tutte le posizioni
func (bp *BybitOrderProcessor) GetPositions(ctx context.Context, symbol string) ([]models.Position, error) {
	// Costruisce l'URL con parametri query
	baseURL := bybitAPIBaseURL + bybitGetPositionsEndpoint

	// Crea i parametri della query
	params := url.Values{}
	params.Set("category", derivativesCategory)
	if symbol != "" {
		params.Set("symbol", symbol)
	} else {
		// Se non è specificato un simbolo, usa USDT come settleCoin per ottenere tutte le posizioni
		params.Set("settleCoin", "USDT")
	}

	// URL completo con parametri
	fullURL := baseURL + "?" + params.Encode()

	// Crea la richiesta HTTP GET
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"

	// Per richieste GET, il payload per la firma è costituito dai parametri query
	queryString := params.Encode()
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, queryString)

	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var positionsResp models.PositionListResponse
	if err := json.Unmarshal(body, &positionsResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Verifica che la richiesta sia andata a buon fine
	if positionsResp.RetCode != 0 {
		return nil, fmt.Errorf("errore API Bybit: %s (codice: %d)", positionsResp.RetMsg, positionsResp.RetCode)
	}

	// Filtra solo le posizioni attive (con size > 0)
	var activePositions []models.Position
	for _, position := range positionsResp.Result.List {
		if position.IsActive() {
			// Aggiungi timestamp di aggiornamento per uso interno
			position.UpdatedAt = time.Unix(positionsResp.Time/1000, 0)
			activePositions = append(activePositions, position)
		}
	}

	return activePositions, nil
}

// CanBeUpdated verifica se un ordine può essere aggiornato basandosi sul suo stato
func (bp *BybitOrderProcessor) CanBeUpdated(orderStatus models.OrderStatus) bool {
	switch orderStatus {
	case models.OrderStatusFilled, models.OrderStatusPartiallyFilled:
		// Solo ordini che hanno creato posizioni possono essere aggiornati
		return true
	case models.OrderStatusNew, models.OrderStatusUntriggered:
		// Ordini non ancora eseguiti non possono essere aggiornati
		return false
	default:
		// Stati come Cancelled, Rejected, ecc. non possono essere aggiornati
		return false
	}
}

// isUUIDFormat verifica se la stringa è in formato UUID (orderID di Bybit)
// o se è in formato personalizzato (orderLinkID nostro)
func isUUIDFormat(id string) bool {
	// UUID format: 8-4-4-4-12 caratteri (es: 550e8400-e29b-41d4-a716-446655440000)
	// OrderLinkID nostro: prefix_symbol_timestamp (es: long_BTCUSDT_1234567890)
	return len(id) == 36 && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-'
}

// generateSignature genera la firma HMAC SHA256 richiesta da Bybit
func (bp *BybitOrderProcessor) generateSignature(timestamp, apiKey, recvWindow, body string) string {
	// Costruisce il payload per la firma
	payload := timestamp + apiKey + recvWindow + body

	// Calcola HMAC SHA256
	h := hmac.New(sha256.New, []byte(bp.apiSecret))
	h.Write([]byte(payload))

	return hex.EncodeToString(h.Sum(nil))
}

// GetWalletBalance recupera il saldo del wallet per un account specifico
// Se coin è vuoto, restituisce tutti i saldi; altrimenti filtra per la criptovaluta specificata
func (bp *BybitOrderProcessor) GetWalletBalance(ctx context.Context, accountType, coin string) (*models.WalletBalanceResponse, error) {
	// Costruisce l'URL con parametri query
	baseURL := bybitAPIBaseURL + bybitGetWalletBalanceEndpoint

	// Crea i parametri della query
	params := url.Values{}
	params.Set("accountType", accountType)
	if coin != "" {
		params.Set("coin", coin)
	}

	// URL completo con parametri
	fullURL := baseURL + "?" + params.Encode()

	// Crea la richiesta HTTP GET
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Aggiungi headers per l'autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"

	// Per richieste GET, il payload per la firma è costituito dai parametri query
	queryString := params.Encode()
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, queryString)

	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Esegui la richiesta
	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta: %w", err)
	}
	defer resp.Body.Close()

	// Leggi la risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Decodifica la risposta
	var walletResp models.WalletBalanceResponse
	if err := json.Unmarshal(body, &walletResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Verifica che la richiesta sia andata a buon fine
	if walletResp.RetCode != 0 {
		return nil, fmt.Errorf("errore API Bybit: %s (codice: %d)", walletResp.RetMsg, walletResp.RetCode)
	}

	return &walletResp, nil
}

// GetUSDTBalance recupera il saldo USDT dal wallet (metodo di convenienza)
func (bp *BybitOrderProcessor) GetUSDTBalance(ctx context.Context) (float64, error) {
	// Usa accountType "UNIFIED" per ottenere il saldo unificato
	walletResp, err := bp.GetWalletBalance(ctx, "UNIFIED", "USDT")
	if err != nil {
		return 0, fmt.Errorf("errore nel recupero saldo USDT: %w", err)
	}

	// Ottieni il saldo USDT
	usdtBalance, found := walletResp.GetCoinBalance("USDT")
	if !found {
		return 0, fmt.Errorf("saldo USDT non trovato")
	}

	// Converte in float64
	balance, err := usdtBalance.GetEquityFloat()
	if err != nil {
		return 0, fmt.Errorf("errore nella conversione del saldo USDT: %w", err)
	}

	return balance, nil
}

// GetCoinBalance recupera il saldo per una specifica criptovaluta (metodo di convenienza)
func (bp *BybitOrderProcessor) GetCoinBalance(ctx context.Context, coin string) (float64, error) {
	// Usa accountType "UNIFIED" per ottenere il saldo unificato
	walletResp, err := bp.GetWalletBalance(ctx, "UNIFIED", coin)
	if err != nil {
		return 0, fmt.Errorf("errore nel recupero saldo %s: %w", coin, err)
	}

	// Ottieni il saldo per la criptovaluta specificata
	coinBalance, found := walletResp.GetCoinBalance(coin)
	if !found {
		return 0, fmt.Errorf("saldo %s non trovato", coin)
	}

	// Converte in float64
	balance, err := coinBalance.GetEquityFloat()
	if err != nil {
		return 0, fmt.Errorf("errore nella conversione del saldo %s: %w", coin, err)
	}

	return balance, nil
}

// GenerateOrderLinkID genera un ID univoco per l'ordine
func GenerateOrderLinkID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
}
