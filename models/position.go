package models

import (
	"strconv"
	"time"
)

// PositionSide rappresenta il lato della posizione
type PositionSide string

const (
	PositionSideBuy  PositionSide = "Buy"  // Long position
	PositionSideSell PositionSide = "Sell" // Short position
)

// PositionMode rappresenta la modalità della posizione
type PositionMode string

const (
	PositionModeOneWay PositionMode = "0" // One-Way Mode
	PositionModeHedge  PositionMode = "3" // Hedge Mode
)

// PositionStatus rappresenta lo stato della posizione
type PositionStatus string

const (
	PositionStatusNormal PositionStatus = "Normal"
	PositionStatusLiq    PositionStatus = "Liq"
	PositionStatusAdl    PositionStatus = "Adl"
)

// Position rappresenta una posizione aperta su Bybit
type Position struct {
	Symbol            string         `json:"symbol"`            // Simbolo trading (es. "DOGEUSDT")
	PositionIdx       int            `json:"positionIdx"`       // 0: One-Way Mode, 1: Buy side of Hedge Mode, 2: Sell side of Hedge Mode
	Side              PositionSide   `json:"side"`              // Buy (Long) o Sell (Short)
	Size              string         `json:"size"`              // Dimensione della posizione
	EntryPrice        string         `json:"entryPrice"`        // Prezzo di entrata
	MarkPrice         string         `json:"markPrice"`         // Prezzo di mark
	UnrealisedPnl     string         `json:"unrealisedPnl"`     // PnL non realizzato
	RealisedPnl       string         `json:"realisedPnl"`       // PnL realizzato
	Leverage          string         `json:"leverage"`          // Leva finanziaria
	IsIsolated        bool           `json:"isIsolated"`        // Se la posizione è isolata
	AutoAddMargin     int            `json:"autoAddMargin"`     // Auto add margin (0=off, 1=on)
	PositionStatus    PositionStatus `json:"positionStatus"`    // Stato della posizione
	PositionBalance   string         `json:"positionBalance"`   // Bilancio della posizione
	UpdatedTime       string         `json:"updatedTime"`       // Timestamp ultimo aggiornamento
	TakeProfit        string         `json:"takeProfit"`        // Prezzo take profit
	StopLoss          string         `json:"stopLoss"`          // Prezzo stop loss
	TpTriggerBy       string         `json:"tpTriggerBy"`       // Trigger per TP
	SlTriggerBy       string         `json:"slTriggerBy"`       // Trigger per SL
	PositionMM        string         `json:"positionMM"`        // Maintenance margin
	PositionIM        string         `json:"positionIM"`        // Initial margin
	RiskID            interface{}    `json:"riskId"`            // Risk ID (può essere string o number)
	RiskLimitValue    string         `json:"riskLimitValue"`    // Risk limit value
	TrailingStop      string         `json:"trailingStop"`      // Trailing stop
	TrailingActive    string         `json:"trailingActive"`    // Trailing active
	TrailingTrigger   string         `json:"trailingTrigger"`   // Trailing trigger
	SessionAvgPrice   string         `json:"sessionAvgPrice"`   // Prezzo medio della sessione
	Delta             string         `json:"delta"`             // Delta
	Gamma             string         `json:"gamma"`             // Gamma
	Vega              string         `json:"vega"`              // Vega
	Theta             string         `json:"theta"`             // Theta
	CurRealisedPnl    string         `json:"curRealisedPnl"`    // PnL realizzato corrente
	CurUnrealisedPnl  string         `json:"curUnrealisedPnl"`  // PnL non realizzato corrente
	PnlRate           string         `json:"pnlRate"`           // Tasso PnL
	CloseOrderID      string         `json:"closeOrderId"`      // ID ordine di chiusura
	OcCalcData        string         `json:"ocCalcData"`        // OC calc data
	OrderMargin       string         `json:"orderMargin"`       // Order margin
	WalletBalance     string         `json:"walletBalance"`     // Bilancio wallet
	CumRealisedPnl    string         `json:"cumRealisedPnl"`    // PnL realizzato cumulativo
	CumFunding        string         `json:"cumFunding"`        // Funding cumulativo
	Notional          string         `json:"notional"`          // Notional value
	UnrealisedPnlPcnt string         `json:"unrealisedPnlPcnt"` // Percentuale PnL non realizzato
	StpId             string         `json:"stpId"`             // STP ID
	StpType           string         `json:"stpType"`           // STP type
	CreatedTime       string         `json:"createdTime"`       // Timestamp creazione
	UpdatedAt         time.Time      `json:"updatedAt"`         // Timestamp aggiornamento per uso interno
}

// PositionListResponse rappresenta la risposta dell'API per la lista delle posizioni
type PositionListResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []Position `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// IsLong verifica se la posizione è long
func (p *Position) IsLong() bool {
	return p.Side == PositionSideBuy
}

// IsShort verifica se la posizione è short
func (p *Position) IsShort() bool {
	return p.Side == PositionSideSell
}

// HasStopLoss verifica se la posizione ha uno stop loss impostato
func (p *Position) HasStopLoss() bool {
	return p.StopLoss != "" && p.StopLoss != "0"
}

// HasTakeProfit verifica se la posizione ha un take profit impostato
func (p *Position) HasTakeProfit() bool {
	return p.TakeProfit != "" && p.TakeProfit != "0"
}

// IsActive verifica se la posizione è attiva (ha una dimensione > 0)
func (p *Position) IsActive() bool {
	return p.GetSizeFloat() > 0
}

// GetSizeFloat restituisce la dimensione come float64
func (p *Position) GetSizeFloat() float64 {
	if p.Size == "" {
		return 0
	}
	value, err := strconv.ParseFloat(p.Size, 64)
	if err != nil {
		return 0
	}
	return value
}

// GetEntryPriceFloat restituisce il prezzo di entrata come float64
func (p *Position) GetEntryPriceFloat() float64 {
	if p.EntryPrice == "" {
		return 0
	}
	value, err := strconv.ParseFloat(p.EntryPrice, 64)
	if err != nil {
		return 0
	}
	return value
}

// GetUnrealisedPnlFloat restituisce il PnL non realizzato come float64
func (p *Position) GetUnrealisedPnlFloat() float64 {
	if p.UnrealisedPnl == "" {
		return 0
	}
	value, err := strconv.ParseFloat(p.UnrealisedPnl, 64)
	if err != nil {
		return 0
	}
	return value
}
