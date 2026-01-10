// Package state provides thread-safe state management for CycleTLS.
// This file handles WebSocket connection tracking.
package state

import (
	"sync"
)

// activeWebSockets stores all active WebSocket connections by their ID.
// Uses interface{} to avoid circular imports - callers type-assert to *WebSocketConnection.
var activeWebSockets = make(map[string]interface{})

// activeWebSocketsMutex protects concurrent access to activeWebSockets.
// Uses RWMutex for read-heavy workload optimization.
var activeWebSocketsMutex sync.RWMutex

// RegisterWebSocket adds a WebSocket connection to the active connections map.
// The connection should be a *WebSocketConnection from the cycletls package.
func RegisterWebSocket(id string, conn interface{}) {
	activeWebSocketsMutex.Lock()
	defer activeWebSocketsMutex.Unlock()
	activeWebSockets[id] = conn
}

// GetWebSocket retrieves a WebSocket connection by its ID.
// Returns the connection and true if found, nil and false otherwise.
// Callers should type-assert the returned interface{} to *WebSocketConnection.
func GetWebSocket(id string) (interface{}, bool) {
	activeWebSocketsMutex.RLock()
	defer activeWebSocketsMutex.RUnlock()
	conn, exists := activeWebSockets[id]
	return conn, exists
}

// UnregisterWebSocket removes a WebSocket connection from the active connections map.
func UnregisterWebSocket(id string) {
	activeWebSocketsMutex.Lock()
	defer activeWebSocketsMutex.Unlock()
	delete(activeWebSockets, id)
}
