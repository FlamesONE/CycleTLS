package state

import (
	"context"
	"sync"
)

// activeRequests maps request IDs to their cancel functions
var activeRequests = make(map[string]context.CancelFunc)

// activeRequestsMutex protects concurrent access to activeRequests
var activeRequestsMutex sync.Mutex

// RegisterRequest adds a request's cancel function to the tracker.
// This allows the request to be cancelled later by its ID.
func RegisterRequest(id string, cancel context.CancelFunc) {
	activeRequestsMutex.Lock()
	defer activeRequestsMutex.Unlock()
	activeRequests[id] = cancel
}

// UnregisterRequest removes a request from the tracker.
// This should be called when a request completes normally.
func UnregisterRequest(id string) {
	activeRequestsMutex.Lock()
	defer activeRequestsMutex.Unlock()
	delete(activeRequests, id)
}

// CancelRequest cancels and removes a request from the tracker.
// Returns true if the request was found and cancelled, false otherwise.
func CancelRequest(id string) bool {
	activeRequestsMutex.Lock()
	defer activeRequestsMutex.Unlock()
	if cancel, exists := activeRequests[id]; exists {
		cancel()
		delete(activeRequests, id)
		return true
	}
	return false
}
