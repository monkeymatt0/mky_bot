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

	// Categoria per mercati derivati perpetual
	derivativesCategory = "linear"
)

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
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
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
	Category    string  `json:"category"`              // "linear" per derivatives
	Symbol      string  `json:"symbol"`                // Es. "BTCUSDT"
	TakeProfit  *string `json:"takeProfit,omitempty"`  // Prezzo take profit come stringa
	StopLoss    *string `json:"stopLoss,omitempty"`    // Prezzo stop loss come stringa
	TpTriggerBy string  `json:"tpTriggerBy,omitempty"` // Trigger per TP: "LastPrice", "IndexPrice", "MarkPrice"
	SlTriggerBy string  `json:"slTriggerBy,omitempty"` // Trigger per SL: "LastPrice", "IndexPrice", "MarkPrice"
	PositionIdx int     `json:"positionIdx"`           // Indice posizione
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
// Usa ordini Stop per comprare quando il prezzo raggiunge il livello specificato
func (bp *BybitOrderProcessor) PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	// Genera un ID univoco per l'ordine
	orderLinkID := fmt.Sprintf("long_%s_%d", symbol, time.Now().Unix())

	// Crea la richiesta di ordine Stop per LONG
	// Ordine Stop Buy: compra quando il prezzo raggiunge o supera il trigger price
	orderReq := models.OrderRequest{
		Category:     derivativesCategory,
		Symbol:       symbol,
		Side:         models.OrderSideBuy,
		OrderType:    models.OrderTypeStop,
		Qty:          strconv.FormatFloat(quantity, 'f', -1, 64),
		TriggerPrice: strconv.FormatFloat(price, 'f', -1, 64),      // Prezzo trigger
		Price:        strconv.FormatFloat(price*1.01, 'f', -1, 64), // Prezzo leggermente sopra per assicurare il fill
		StopLoss:     strconv.FormatFloat(stopLoss, 'f', -1, 64),
		TakeProfit:   strconv.FormatFloat(takeProfit, 'f', -1, 64),
		TimeInForce:  models.TimeInForceGTC,
		OrderLinkId:  orderLinkID,
	}

	return bp.placeOrder(ctx, &orderReq)
}

// PlaceShortOrder implementa l'interfaccia OrderProcessor per ordini short
// Usa ordini Stop per vendere quando il prezzo raggiunge il livello specificato
func (bp *BybitOrderProcessor) PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	// Genera un ID univoco per l'ordine
	orderLinkID := fmt.Sprintf("short_%s_%d", symbol, time.Now().Unix())

	// Crea la richiesta di ordine Stop per SHORT
	// Ordine Stop Sell: vende quando il prezzo raggiunge o scende sotto il trigger price
	orderReq := models.OrderRequest{
		Category:     derivativesCategory,
		Symbol:       symbol,
		Side:         models.OrderSideSell,
		OrderType:    models.OrderTypeStop,
		Qty:          strconv.FormatFloat(quantity, 'f', -1, 64),
		TriggerPrice: strconv.FormatFloat(price, 'f', -1, 64),      // Prezzo trigger
		Price:        strconv.FormatFloat(price*0.99, 'f', -1, 64), // Prezzo leggermente sotto per assicurare il fill
		StopLoss:     strconv.FormatFloat(stopLoss, 'f', -1, 64),
		TakeProfit:   strconv.FormatFloat(takeProfit, 'f', -1, 64),
		TimeInForce:  models.TimeInForceGTC,
		OrderLinkId:  orderLinkID,
	}

	return bp.placeOrder(ctx, &orderReq)
}

// placeOrder invia l'ordine a Bybit usando le API autenticate
func (bp *BybitOrderProcessor) placeOrder(ctx context.Context, orderReq *models.OrderRequest) (*models.OrderResponse, error) {
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
		stopLossStr := strconv.FormatFloat(*params.StopLoss, 'f', 2, 64)
		updateReq.StopLoss = &stopLossStr
	}

	// Converte TakeProfit in stringa se specificato
	if params.TakeProfit != nil {
		takeProfitStr := strconv.FormatFloat(*params.TakeProfit, 'f', 2, 64)
		updateReq.TakeProfit = &takeProfitStr
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

// GenerateOrderLinkID genera un ID univoco per l'ordine
func GenerateOrderLinkID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
}
