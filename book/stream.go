package book

import (
	"context"

	"cross-exchange-arbitrage/models"
)

// OrderBookStreamer defines the interface for streaming orderbook data
type OrderBookStreamer interface {
	// OrderBookStream opens a websocket connection to stream orderbook data for a specific symbol
	// ctx is used to control the lifecycle of the stream
	// symbol is the trading pair to stream (e.g. "BTCUSDT")
	// depth is the number of orderbook levels to maintain (optional, depends on exchange support)
	// updateChan is the channel where orderbook updates will be sent
	// errChan is the channel where any errors will be sent
	OrderBookStream(
		ctx context.Context,
		symbol string,
		depth int,
		updateChan chan<- *models.OrderBookData,
		errChan chan<- error,
	) error
}
