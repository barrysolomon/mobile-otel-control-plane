package push

import (
	"sync"

	"github.com/gorilla/websocket"
)

// ConnectionEntry tracks a single WebSocket connection.
type ConnectionEntry struct {
	DeviceID   string
	Conn       *websocket.Conn
	Attributes map[string]string
}

// ConnectionRegistry manages all active WebSocket connections.
type ConnectionRegistry struct {
	mu    sync.RWMutex
	conns map[string]*ConnectionEntry
}

// NewConnectionRegistry creates an empty connection registry.
func NewConnectionRegistry() *ConnectionRegistry {
	return &ConnectionRegistry{
		conns: make(map[string]*ConnectionEntry),
	}
}

// Register adds a device connection to the registry.
func (r *ConnectionRegistry) Register(deviceID string, conn *websocket.Conn, attrs map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[deviceID] = &ConnectionEntry{
		DeviceID:   deviceID,
		Conn:       conn,
		Attributes: attrs,
	}
}

// Unregister removes a device connection.
func (r *ConnectionRegistry) Unregister(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, deviceID)
}

// GetConnection returns the connection entry for a device.
func (r *ConnectionRegistry) GetConnection(deviceID string) (*ConnectionEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.conns[deviceID]
	return entry, ok
}

// ConnectedCount returns the number of active connections.
func (r *ConnectionRegistry) ConnectedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.conns)
}

// GetConnectedDeviceIDs filters a list of device IDs to only those with active connections.
func (r *ConnectionRegistry) GetConnectedDeviceIDs(deviceIDs []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connected []string
	for _, id := range deviceIDs {
		if _, ok := r.conns[id]; ok {
			connected = append(connected, id)
		}
	}
	return connected
}

// AllConnectedIDs returns all connected device IDs.
func (r *ConnectionRegistry) AllConnectedIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.conns))
	for id := range r.conns {
		ids = append(ids, id)
	}
	return ids
}
