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
func (bp *BybitTestnetOrderProcessor) PlaceLongOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	orderLinkID := fmt.Sprintf("testnet_long_%s_%d", symbol, time.Now().Unix())

	orderReq := models.OrderRequest{
		Category:     derivativesCategory,
		Symbol:       symbol,
		Side:         models.OrderSideBuy,
		OrderType:    models.OrderTypeStop,
		Qty:          strconv.FormatFloat(quantity, 'f', -1, 64),
		TriggerPrice: strconv.FormatFloat(price, 'f', -1, 64),
		Price:        strconv.FormatFloat(price*1.01, 'f', -1, 64),
		StopLoss:     strconv.FormatFloat(stopLoss, 'f', -1, 64),
		TakeProfit:   strconv.FormatFloat(takeProfit, 'f', -1, 64),
		TimeInForce:  models.TimeInForceGTC,
		OrderLinkId:  orderLinkID,
	}

	return bp.placeOrder(ctx, &orderReq)
}

// PlaceShortOrder implementa l'interfaccia OrderProcessor per ordini short su testnet
func (bp *BybitTestnetOrderProcessor) PlaceShortOrder(ctx context.Context, symbol string, price, quantity, stopLoss, takeProfit float64) (*models.OrderResponse, error) {
	orderLinkID := fmt.Sprintf("testnet_short_%s_%d", symbol, time.Now().Unix())

	orderReq := models.OrderRequest{
		Category:     derivativesCategory,
		Symbol:       symbol,
		Side:         models.OrderSideSell,
		OrderType:    models.OrderTypeStop,
		Qty:          strconv.FormatFloat(quantity, 'f', -1, 64),
		TriggerPrice: strconv.FormatFloat(price, 'f', -1, 64),
		Price:        strconv.FormatFloat(price*0.99, 'f', -1, 64),
		StopLoss:     strconv.FormatFloat(stopLoss, 'f', -1, 64),
		TakeProfit:   strconv.FormatFloat(takeProfit, 'f', -1, 64),
		TimeInForce:  models.TimeInForceGTC,
		OrderLinkId:  orderLinkID,
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

// generateSignature genera la firma HMAC SHA256
func (bp *BybitTestnetOrderProcessor) generateSignature(timestamp, apiKey, recvWindow, body string) string {
	payload := timestamp + apiKey + recvWindow + body
	h := hmac.New(sha256.New, []byte(bp.apiSecret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
