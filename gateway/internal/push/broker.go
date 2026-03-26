package push

import (
	"encoding/json"
	"log"
	"sort"
	"time"

	"github.com/gorilla/websocket"
)

// Broker orchestrates push delivery across channels.
type Broker struct {
	registry *ConnectionRegistry
}

// NewBroker creates a push broker with the given connection registry.
func NewBroker(registry *ConnectionRegistry) *Broker {
	return &Broker{registry: registry}
}

// Registry returns the broker's connection registry.
func (b *Broker) Registry() *ConnectionRegistry {
	return b.registry
}

// Deliver sends a fleet alert to a set of target device IDs.
func (b *Broker) Deliver(alert interface{}, targetDeviceIDs []string) []DeliveryResult {
	payload, err := json.Marshal(alert)
	if err != nil {
		log.Printf("ERROR: marshal alert: %v", err)
		return nil
	}

	results := make([]DeliveryResult, 0, len(targetDeviceIDs))

	for _, deviceID := range targetDeviceIDs {
		entry, connected := b.registry.GetConnection(deviceID)
		if connected && entry.Conn != nil {
			err := entry.Conn.WriteMessage(websocket.TextMessage, payload)
			if err != nil {
				results = append(results, DeliveryResult{
					DeviceID: deviceID, Channel: "websocket", Success: false, Error: err.Error(),
				})
			} else {
				results = append(results, DeliveryResult{
					DeviceID: deviceID, Channel: "websocket", Success: true,
				})
			}
		} else {
			results = append(results, DeliveryResult{
				DeviceID: deviceID, Channel: "poll", Success: true,
			})
		}
	}

	return results
}

// EstimateDelivery returns delivery estimates for a set of devices.
func (b *Broker) EstimateDelivery(deviceIDs []string) DeliveryEstimate {
	wsDevices := b.registry.GetConnectedDeviceIDs(deviceIDs)
	pollDevices := len(deviceIDs) - len(wsDevices)

	return DeliveryEstimate{
		ImmediateDelivery: len(wsDevices),
		DelayedDelivery:   pollDevices,
		MaxDeliveryTime:   60 * time.Second,
	}
}

// SortByPriority sorts alerts by priority (critical first).
func SortByPriority(alerts []PrioritizedAlert) {
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Priority < alerts[j].Priority
	})
}

// PrioritizedAlert wraps an alert with its priority for sorting.
type PrioritizedAlert struct {
	Alert    interface{}
	Priority int
}
