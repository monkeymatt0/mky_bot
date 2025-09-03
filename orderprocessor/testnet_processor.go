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
)

const (
	// URL di base per le API REST di Bybit (testnet)
	testnetAPIBaseURL = "https://api-testnet.bybit.com"
)

// BybitTestnetOrderProcessor implementa OrderProcessor per Bybit Testnet
type BybitTestnetOrderProcessor struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewBybitTestnetOrderProcessor crea una nuova istanza per testnet
func NewBybitTestnetOrderProcessor(apiKey, apiSecret string) *BybitTestnetOrderProcessor {
	return &BybitTestnetOrderProcessor{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PlaceLongOrder implementa l'interfaccia OrderProcessor per ordini long su testnet
// Crea un ordine Stop-Limit: si attiva al trigger price e poi esegue un ordine limit al prezzo specificato
func (bp *BybitTestnetOrderProcessor) PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	orderLinkID := fmt.Sprintf("testnet_long_%s_%d", symbol, time.Now().Unix())

	// Per ordini LONG Stop-Limit:
	// - TriggerPrice: prezzo a cui si attiva l'ordine
	// - Price: prezzo limite a cui vogliamo comprare (leggermente sopra il trigger per garantire fill)
	limitPrice := price * 1.002 // Prezzo limite 0.2% sopra il trigger per assicurare esecuzione

	orderReq := models.OrderRequest{
		Category:         derivativesCategory,
		Symbol:           symbol,
		Side:             models.OrderSideBuy,
		OrderType:        models.OrderTypeLimit, // Ordine Limit condizionale
		Qty:              strconv.FormatFloat(quantity, 'f', -1, 64),
		Price:            strconv.FormatFloat(limitPrice, 'f', 2, 64), // Prezzo limite fisso
		TriggerPrice:     strconv.FormatFloat(price, 'f', 2, 64),      // Prezzo trigger
		TriggerDirection: models.TriggerDirectionRising,               // Per Long: trigger quando prezzo sale
		StopLoss:         strconv.FormatFloat(stopLoss, 'f', 2, 64),
		TakeProfit:       strconv.FormatFloat(takeProfit, 'f', 2, 64),
		TimeInForce:      models.TimeInForceGTC,
		OrderLinkId:      orderLinkID,
	}

	return bp.placeOrder(ctx, &orderReq)
}

// PlaceShortOrder implementa l'interfaccia OrderProcessor per ordini short su testnet
// Crea un ordine Stop-Limit: si attiva al trigger price e poi esegue un ordine limit al prezzo specificato
func (bp *BybitTestnetOrderProcessor) PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	orderLinkID := fmt.Sprintf("testnet_short_%s_%d", symbol, time.Now().Unix())

	// Per ordini SHORT Stop-Limit:
	// - TriggerPrice: prezzo a cui si attiva l'ordine
	// - Price: prezzo limite a cui vogliamo vendere (leggermente sotto il trigger per garantire fill)
	limitPrice := price * 0.998 // Prezzo limite 0.2% sotto il trigger per assicurare esecuzione

	orderReq := models.OrderRequest{
		Category:         derivativesCategory,
		Symbol:           symbol,
		Side:             models.OrderSideSell,
		OrderType:        models.OrderTypeLimit, // Ordine Limit condizionale
		Qty:              strconv.FormatFloat(quantity, 'f', -1, 64),
		Price:            strconv.FormatFloat(limitPrice, 'f', 2, 64), // Prezzo limite fisso
		TriggerPrice:     strconv.FormatFloat(price, 'f', 2, 64),      // Prezzo trigger
		TriggerDirection: models.TriggerDirectionFalling,              // Per Short: trigger quando prezzo scende
		StopLoss:         strconv.FormatFloat(stopLoss, 'f', 2, 64),
		TakeProfit:       strconv.FormatFloat(takeProfit, 'f', 2, 64),
		TimeInForce:      models.TimeInForceGTC,
		OrderLinkId:      orderLinkID,
	}

	return bp.placeOrder(ctx, &orderReq)
}

// placeOrder invia l'ordine a Bybit Testnet
func (bp *BybitTestnetOrderProcessor) placeOrder(ctx context.Context, orderReq *models.OrderRequest) (*models.OrderResponse, error) {
	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		return nil, fmt.Errorf("errore nella serializzazione dell'ordine: %w", err)
	}

	// Usa l'URL della testnet
	url := testnetAPIBaseURL + bybitPlaceOrderEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Headers per autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Log per debug
	fmt.Printf("🔗 Sending request to: %s\n", url)
	fmt.Printf("📦 Request body: %s\n", string(jsonData))

	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("errore nell'esecuzione della richiesta: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della risposta: %w", err)
	}

	// Log della risposta per debug
	fmt.Printf("📨 Response status: %d\n", resp.StatusCode)
	fmt.Printf("📨 Response body: %s\n", string(body))

	var apiResp BybitAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Converte la risposta
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

	// Converte i valori
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

	if apiResp.RetCode == 0 {
		orderResp.Status = models.OrderStatusUntriggered
	} else {
		orderResp.Status = models.OrderStatusRejected
	}

	return orderResp, nil
}

// DeleteOrder cancella un ordine esistente su testnet usando l'orderID o orderLinkID
func (bp *BybitTestnetOrderProcessor) DeleteOrder(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error) {
	// Crea la richiesta di cancellazione
	cancelReq := struct {
		Category    string `json:"category"`
		Symbol      string `json:"symbol"`
		OrderID     string `json:"orderId,omitempty"`
		OrderLinkID string `json:"orderLinkId,omitempty"`
	}{
		Category: derivativesCategory,
		Symbol:   symbol,
	}

	// Determina se è un orderID (UUID format) o orderLinkID (nostro formato personalizzato)
	if isTestnetUUIDFormat(orderID) {
		cancelReq.OrderID = orderID
	} else {
		cancelReq.OrderLinkID = orderID
	}

	// Serializza la richiesta in JSON
	jsonData, err := json.Marshal(cancelReq)
	if err != nil {
		return nil, fmt.Errorf("errore nella serializzazione della cancellazione: %w", err)
	}

	// Usa l'URL della testnet per cancellazione
	url := testnetAPIBaseURL + "/v5/order/cancel"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Headers per autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Log per debug
	fmt.Printf("🗑️ Cancelling order: %s\n", url)
	fmt.Printf("📦 Cancel request: %s\n", string(jsonData))

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

	// Log della risposta per debug
	fmt.Printf("📨 Cancel response status: %d\n", resp.StatusCode)
	fmt.Printf("📨 Cancel response body: %s\n", string(body))

	// Decodifica la risposta
	var cancelResp BybitAPIResponse
	if err := json.Unmarshal(body, &cancelResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Converte la risposta nel formato interno
	orderResp := &models.OrderResponse{
		OrderID:      cancelResp.Result.OrderID,
		OrderLinkID:  cancelResp.Result.OrderLinkID,
		Symbol:       symbol,
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

// UpdateOrder aggiorna stop loss e/o take profit di una posizione esistente su testnet
func (bp *BybitTestnetOrderProcessor) UpdateOrder(ctx context.Context, params UpdateOrderParams) (*models.OrderResponse, error) {
	// Valida che almeno uno tra StopLoss e TakeProfit sia specificato
	if params.StopLoss == nil && params.TakeProfit == nil {
		return nil, fmt.Errorf("almeno uno tra StopLoss e TakeProfit deve essere specificato")
	}

	// Crea la richiesta di aggiornamento per testnet
	updateReq := struct {
		Category    string  `json:"category"`
		Symbol      string  `json:"symbol"`
		TakeProfit  *string `json:"takeProfit,omitempty"`
		StopLoss    *string `json:"stopLoss,omitempty"`
		TpTriggerBy string  `json:"tpTriggerBy,omitempty"`
		SlTriggerBy string  `json:"slTriggerBy,omitempty"`
		PositionIdx int     `json:"positionIdx"`
	}{
		Category:    derivativesCategory,
		Symbol:      params.Symbol,
		PositionIdx: params.PositionIdx,
		TpTriggerBy: "LastPrice",
		SlTriggerBy: "LastPrice",
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

	// Usa l'URL della testnet per aggiornamento
	url := testnetAPIBaseURL + "/v5/position/trading-stop"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("errore nella creazione della richiesta HTTP: %w", err)
	}

	// Headers per autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, string(jsonData))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Log per debug
	fmt.Printf("🔄 Updating order: %s\n", url)
	fmt.Printf("📦 Update request: %s\n", string(jsonData))

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

	// Log della risposta per debug
	fmt.Printf("📨 Update response status: %d\n", resp.StatusCode)
	fmt.Printf("📨 Update response body: %s\n", string(body))

	// Decodifica la risposta
	var updateResp BybitAPIResponse
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

// GetOrderStatus recupera lo stato di un ordine specifico su testnet
func (bp *BybitTestnetOrderProcessor) GetOrderStatus(ctx context.Context, symbol, orderID string) (*models.OrderResponse, error) {
	// Costruisce l'URL con parametri query per testnet
	baseURL := testnetAPIBaseURL + "/v5/order/realtime"

	// Crea i parametri della query
	params := url.Values{}
	params.Set("category", derivativesCategory)
	params.Set("symbol", symbol)

	// Determina se è un orderID (UUID format) o orderLinkID (nostro formato personalizzato)
	if isTestnetUUIDFormat(orderID) {
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

	// Headers per autenticazione
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recv_window := "5000"

	// Per richieste GET, il payload per la firma è costituito dai parametri query
	queryString := params.Encode()
	signature := bp.generateSignature(timestamp, bp.apiKey, recv_window, queryString)

	req.Header.Set("X-BAPI-API-KEY", bp.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recv_window)
	req.Header.Set("X-BAPI-SIGN", signature)

	// Log per debug
	fmt.Printf("🔍 Getting order status: %s\n", fullURL)

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

	// Log della risposta per debug
	fmt.Printf("📨 Status response status: %d\n", resp.StatusCode)
	fmt.Printf("📨 Status response body: %s\n", string(body))

	// Decodifica la risposta usando la stessa struttura del mainnet
	var statusResp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OrderID     string `json:"orderId"`
				OrderLinkID string `json:"orderLinkId"`
				Symbol      string `json:"symbol"`
				OrderStatus string `json:"orderStatus"`
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

	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("errore nella decodifica della risposta: %w", err)
	}

	// Verifica che la richiesta sia andata a buon fine
	if statusResp.RetCode != 0 {
		return nil, fmt.Errorf("errore API Bybit Testnet: %s (codice: %d)", statusResp.RetMsg, statusResp.RetCode)
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

// CanBeUpdated verifica se un ordine può essere aggiornato basandosi sul suo stato (testnet)
func (bp *BybitTestnetOrderProcessor) CanBeUpdated(orderStatus models.OrderStatus) bool {
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

// isTestnetUUIDFormat verifica se la stringa è in formato UUID
func isTestnetUUIDFormat(id string) bool {
	// UUID format: 8-4-4-4-12 caratteri (es: 550e8400-e29b-41d4-a716-446655440000)
	// OrderLinkID nostro: testnet_prefix_symbol_timestamp (es: testnet_long_BTCUSDT_1234567890)
	return len(id) == 36 && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-'
}

// generateSignature genera la firma HMAC SHA256
func (bp *BybitTestnetOrderProcessor) generateSignature(timestamp, apiKey, recvWindow, body string) string {
	payload := timestamp + apiKey + recvWindow + body
	h := hmac.New(sha256.New, []byte(bp.apiSecret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
