package push

import (
	"time"
)

// DevicePushState tracks the push channel state for a single device.
type DevicePushState struct {
	DeviceID          string     `json:"device_id"`
	WSConnected       bool       `json:"ws_connected"`
	WSConnectedAt     *time.Time `json:"ws_connected_at,omitempty"`
	FCMToken          string     `json:"fcm_token,omitempty"`
	FCMTokenUpdatedAt *time.Time `json:"fcm_token_updated_at,omitempty"`
	LastPushChannel   string     `json:"last_push_channel,omitempty"` // websocket, fcm, poll
	AttributesJSON    string     `json:"attributes_json,omitempty"`
}

// DeliveryEstimate reports expected delivery characteristics.
type DeliveryEstimate struct {
	ImmediateDelivery int           `json:"immediate_delivery"` // WebSocket
	ProbableDelivery  int           `json:"probable_delivery"`  // FCM (~75%)
	DelayedDelivery   int           `json:"delayed_delivery"`   // Poll fallback
	MaxDeliveryTime   time.Duration `json:"max_delivery_time"`
}

// DeliveryResult tracks what happened for a push attempt.
type DeliveryResult struct {
	DeviceID string `json:"device_id"`
	Channel  string `json:"channel"` // websocket, poll
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}
