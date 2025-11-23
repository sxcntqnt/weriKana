// internal/bank/handlers/fill_broadcast.go
// BroadcastFill publishes execution fills to NATS for real-time client updates.

package bank

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// FillEvent is the structure for broadcasting fills.
type FillEvent struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Qty    int     `json:"qty"`
	Status string  `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// BroadcastFill publishes a fill event to NATS subject "fills.broadcast".
// Assumes nc is a global or injected; for now, using a passed *nats.Conn.
func BroadcastFill(nc *nats.Conn, symbol string, price float64, qty int, status string) error {
	event := FillEvent{
		Symbol: symbol,
		Price:  price,
		Qty:    qty,
		Status: status,
		Timestamp: time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("fill.marshal_failed", "err", err)
		return err
	}
	err = nc.Publish("fills.broadcast", data)
	if err != nil {
		slog.Error("fill.publish_failed", "err", err)
		return err
	}
	slog.Info("fill_broadcasted", "symbol", symbol, "price", price, "qty", qty)
	return nil
}
