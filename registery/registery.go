package registery

import (
	"errors"
	"sync"
)

// Registry uses type parameters at struct level instead of method level
type Registry[T any] struct {
	items map[string]T
	mu    sync.RWMutex
}

func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{items: make(map[string]T)}
}

func (r *Registry[T]) Register(key string, value T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = value
}

func (r *Registry[T]) Delete(key string) {
	r.mu.Lock()         // Acquire write lock (since we're modifying the map)
	defer r.mu.Unlock() // Ensure the lock is released

	//if _, exists := r.items[key]; !exists {
	//	return fmt.Errorf("key '%s' not found", key)
	//}

	delete(r.items, key) // Remove the key from the map
	//return nil
}

func (r *Registry[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[string]T) // Reinitialize the map
}

func (r *Registry[T]) Get(key string) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, exists := r.items[key]
	if !exists {
		var zero T
		return zero, errors.New("key not found")
	}
	return val, nil
}

func (r *Registry[T]) Update(key string, newValue T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = newValue
}

func (r *Registry[T]) GetAllValues() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]T, 0, len(r.items))
	for _, v := range r.items {
		result = append(result, v)
	}
	return result
}

func (r *Registry[T]) IsEmpty() bool {
	r.mu.RLock()         // Acquire read lock for thread safety
	defer r.mu.RUnlock() // Ensure the lock is released

	return len(r.items) == 0
}
