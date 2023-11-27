package helper

import (
	"context"
	"sync"
)

type WaitKey struct {
	Value   string
	Timeout context.Context
}

// ConcurrentMap encapsulates a map and a mutex for concurrent access
type ConcurrentMap struct {
	mu   sync.Mutex
	data map[string]WaitKey
}

// NewConcurrentMap creates a new instance of ConcurrentMap
func NewConcurrentMap() *ConcurrentMap {
	return &ConcurrentMap{
		data: make(map[string]WaitKey),
	}
}

// AddKey adds a key-value pair to the map
func (cm *ConcurrentMap) AddKey(key string, value WaitKey) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.data[key] = value
}

// DeleteKey deletes a key from the map
func (cm *ConcurrentMap) DeleteKey(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.data, key)
}

// ReadFromMap read the contents of the map
func (cm *ConcurrentMap) ReadFromMap(key string) WaitKey {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.data[key]
}
