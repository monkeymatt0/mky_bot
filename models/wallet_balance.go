package models

import (
	"strconv"
	"time"
)

// WalletBalance rappresenta il saldo del wallet per una specifica criptovaluta
type WalletBalance struct {
	Coin                string    `json:"coin"`                // Simbolo della criptovaluta (es. "USDT", "BTC")
	Equity              string    `json:"equity"`              // Equity totale
	WalletBalance       string    `json:"walletBalance"`       // Bilancio del wallet
	AvailableToWithdraw string    `json:"availableToWithdraw"` // Disponibile per prelievo
	UpdatedAt           time.Time `json:"updated_at"`          // Timestamp di aggiornamento (aggiunto internamente)
}

// GetEquityFloat restituisce l'equity come float64
func (wb *WalletBalance) GetEquityFloat() (float64, error) {
	return strconv.ParseFloat(wb.Equity, 64)
}

// GetWalletBalanceFloat restituisce il wallet balance come float64
func (wb *WalletBalance) GetWalletBalanceFloat() (float64, error) {
	return strconv.ParseFloat(wb.WalletBalance, 64)
}

// GetAvailableToWithdrawFloat restituisce l'importo disponibile per prelievo come float64
func (wb *WalletBalance) GetAvailableToWithdrawFloat() (float64, error) {
	return strconv.ParseFloat(wb.AvailableToWithdraw, 64)
}

// IsActive verifica se il wallet ha un saldo attivo
func (wb *WalletBalance) IsActive() bool {
	equity, err := wb.GetEquityFloat()
	if err != nil {
		return false
	}
	return equity > 0
}

// AccountInfo rappresenta le informazioni dell'account
type AccountInfo struct {
	AccountType           string          `json:"accountType"`           // Tipo di account (UNIFIED, CONTRACT, SPOT)
	TotalEquity           string          `json:"totalEquity"`           // Equity totale dell'account
	TotalWalletBalance    string          `json:"totalWalletBalance"`    // Bilancio totale del wallet
	TotalMarginBalance    string          `json:"totalMarginBalance"`    // Bilancio totale del margin
	TotalAvailableBalance string          `json:"totalAvailableBalance"` // Bilancio totale disponibile
	Coins                 []WalletBalance `json:"coin"`                  // Lista delle criptovalute
	UpdatedAt             time.Time       `json:"updated_at"`            // Timestamp di aggiornamento (aggiunto internamente)
}

// GetTotalEquityFloat restituisce l'equity totale come float64
func (ai *AccountInfo) GetTotalEquityFloat() (float64, error) {
	return strconv.ParseFloat(ai.TotalEquity, 64)
}

// GetTotalWalletBalanceFloat restituisce il bilancio totale del wallet come float64
func (ai *AccountInfo) GetTotalWalletBalanceFloat() (float64, error) {
	return strconv.ParseFloat(ai.TotalWalletBalance, 64)
}

// GetTotalAvailableBalanceFloat restituisce il bilancio totale disponibile come float64
func (ai *AccountInfo) GetTotalAvailableBalanceFloat() (float64, error) {
	return strconv.ParseFloat(ai.TotalAvailableBalance, 64)
}

// GetCoinBalance cerca il saldo per una specifica criptovaluta
func (ai *AccountInfo) GetCoinBalance(coin string) (*WalletBalance, bool) {
	for _, balance := range ai.Coins {
		if balance.Coin == coin {
			return &balance, true
		}
	}
	return nil, false
}

// GetActiveCoins restituisce solo le criptovalute con saldo attivo
func (ai *AccountInfo) GetActiveCoins() []WalletBalance {
	var activeCoins []WalletBalance
	for _, balance := range ai.Coins {
		if balance.IsActive() {
			activeCoins = append(activeCoins, balance)
		}
	}
	return activeCoins
}

// WalletBalanceResponse rappresenta la risposta completa dell'API Bybit per il wallet balance
type WalletBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []AccountInfo `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// IsSuccess verifica se la richiesta è andata a buon fine
func (wbr *WalletBalanceResponse) IsSuccess() bool {
	return wbr.RetCode == 0
}

// GetFirstAccount restituisce il primo account dalla lista (di solito c'è solo uno)
func (wbr *WalletBalanceResponse) GetFirstAccount() *AccountInfo {
	if len(wbr.Result.List) > 0 {
		account := wbr.Result.List[0]
		account.UpdatedAt = time.Unix(wbr.Time/1000, 0)
		// Aggiorna anche i timestamp delle singole criptovalute
		for i := range account.Coins {
			account.Coins[i].UpdatedAt = account.UpdatedAt
		}
		return &account
	}
	return nil
}

// GetCoinBalance cerca il saldo per una specifica criptovaluta nel primo account
func (wbr *WalletBalanceResponse) GetCoinBalance(coin string) (*WalletBalance, bool) {
	account := wbr.GetFirstAccount()
	if account == nil {
		return nil, false
	}
	return account.GetCoinBalance(coin)
}
