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
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	// URL di base per le API REST di Bybit
	bybitAPIBaseURL = "https://api.bybit.com"

	// Endpoint per piazzare ordini
	bybitPlaceOrderEndpoint = "/v5/order/create"

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
