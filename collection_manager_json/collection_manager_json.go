package collection_manager_json

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// collectionItem is the interface that every item in the collection must implement.
// The ID is of type uuid.UUID for type safety.
type collectionItem interface {
	SetID(uuid.UUID)
	GetID() uuid.UUID
}

// ---
// registry section: in-memory data store.
// This registry uses string keys, which is fine as uuid.UUID is always converted to string for storage.

type registry[T any] struct {
	items map[uuid.UUID]T
	mu    sync.RWMutex
}

func newRegistry[T any]() *registry[T] {
	return &registry[T]{items: make(map[uuid.UUID]T)}
}

func (r *registry[T]) create(key uuid.UUID, value T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = value
}

func (r *registry[T]) read(key uuid.UUID) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, exists := r.items[key]
	if !exists {
		var zero T
		return zero, errors.New("key not found")
	}
	return val, nil
}

func (r *registry[T]) readAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]T, 0, len(r.items))
	for _, v := range r.items {
		result = append(result, v)
	}
	return result
}

func (r *registry[T]) update(key uuid.UUID, newValue T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = newValue
}

func (r *registry[T]) delete(key uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, key)
}

// Count returns the number of items in the registry.
func (r *registry[T]) count() int {
	// Acquire a read lock to safely read the map.
	r.mu.RLock()
	// The defer statement ensures to unlock happens before the function returns.
	defer r.mu.RUnlock()
	return len(r.items)
}

func (r *registry[T]) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[uuid.UUID]T)
}

func (r *registry[T]) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items) == 0
}

// ---

// Manager is the main struct for managing the collection.
type Manager[T collectionItem] struct {
	baseDir string
	items   *registry[T]
	mu      sync.RWMutex
	// fileMutexes is a map of mutexes, one for each file.
	// The key is the item's UUID.
	fileMutexes map[uuid.UUID]*sync.Mutex
}

// New creates a new instance of Manager.
func New[T collectionItem](path string) (*Manager[T], error) {

	if strings.HasSuffix(path, ".json") {
		return nil, errors.New("path must be a directory, not a file")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	manager := &Manager[T]{
		baseDir:     path,
		items:       newRegistry[T](),
		fileMutexes: make(map[uuid.UUID]*sync.Mutex),
	}

	items, err := manager.readAllItemsFromDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}

	for _, item := range items {
		manager.items.create(item.GetID(), item)
	}

	return manager, nil
}

// itemPath returns the path to a JSON file based on the item's ID.
func (m *Manager[T]) itemPath(id uuid.UUID) string {
	return filepath.Join(m.baseDir, id.String()+".json")
}

// getOrCreateMutex returns a mutex for a given item ID, creating it if it doesn't exist.
func (m *Manager[T]) getOrCreateMutex(id uuid.UUID) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.fileMutexes[id]; !ok {
		m.fileMutexes[id] = &sync.Mutex{}
	}
	return m.fileMutexes[id]
}

// deleteMutex removes a mutex from the map after its item has been deleted.
func (m *Manager[T]) deleteMutex(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.fileMutexes, id)
}

// ---
// Helper functions that access the disk directly and are called by public methods.
// They do not have their own locks.

func (m *Manager[T]) readAllItemsFromDisk() ([]T, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}

	items := make([]T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filename := strings.TrimSuffix(entry.Name(), ".json")
		if _, err := uuid.Parse(filename); err != nil {
			fmt.Printf("collection_manager_json: skipping file with invalid UUID filename: %s, error: %v\n", entry.Name(), err)
			continue
		}

		item, err := m.readItemFromDisk(filename)
		if err != nil {
			fmt.Printf("collection_manager_json: error reading item %s: %v\n", filename, err)
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (m *Manager[T]) readItemFromDisk(id string) (T, error) {
	var zero T
	path := filepath.Join(m.baseDir, id+".json")
	file, err := os.ReadFile(path)
	if err != nil {
		return zero, err
	}

	if len(file) == 0 {
		return zero, errors.New("empty file")
	}

	var item T
	if err := json.Unmarshal(file, &item); err != nil {
		return zero, err
	}
	return item, nil
}

func (m *Manager[T]) writeItemToDisk(item T) error {
	path := m.itemPath(item.GetID())
	jsonData, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}

	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return err
	}
	return os.Rename(tempFile, path)
}

// ---
// Public methods that manage concurrency and call helper functions.

// Create a new item in the collection.
func (m *Manager[T]) Create(newItem T) (T, error) {
	mutex := m.getOrCreateMutex(newItem.GetID())
	mutex.Lock()
	defer mutex.Unlock()

	if reflect.ValueOf(newItem).IsNil() {
		var zero T
		return zero, errors.New("cannot create nil item")
	}

	if _, err := m.items.read(newItem.GetID()); err == nil {
		var zero T
		return zero, fmt.Errorf("item with ID %s already exists", newItem.GetID().String())
	}

	if err := m.writeItemToDisk(newItem); err != nil {
		var zero T
		return zero, err
	}

	m.items.create(newItem.GetID(), newItem)
	return newItem, nil
}

// Read an item from the collection by its ID.
func (m *Manager[T]) Read(id uuid.UUID) (T, error) {
	mutex := m.getOrCreateMutex(id)
	mutex.Lock()
	defer mutex.Unlock()
	// NOTE: The read operation could use an RWMutex for better performance,
	// but for simplicity, we use a Mutex here.
	return m.items.read(id)
}

// ReadAll items from the collection.
func (m *Manager[T]) ReadAll() ([]T, error) {
	// NOTE: This operation still needs a coarse-grained lock or a more complex solution
	// to prevent items from being created/deleted while iterating.
	// For now, we will use the registry's lock which is a coarse-grained lock.
	return m.items.readAll(), nil
}

// Update an existing item in the collection.
func (m *Manager[T]) Update(updatedItem T) (T, error) {
	mutex := m.getOrCreateMutex(updatedItem.GetID())
	mutex.Lock()
	defer mutex.Unlock()

	if reflect.ValueOf(updatedItem).IsNil() {
		var zero T
		return zero, errors.New("cannot update with nil item")
	}

	if _, err := m.items.read(updatedItem.GetID()); err != nil {
		var zero T
		return zero, fmt.Errorf("item with ID %s does not exist", updatedItem.GetID().String())
	}

	if err := m.writeItemToDisk(updatedItem); err != nil {
		var zero T
		return zero, err
	}

	m.items.update(updatedItem.GetID(), updatedItem)
	return updatedItem, nil
}

// Delete an item from the collection by its ID.
func (m *Manager[T]) Delete(id uuid.UUID) error {
	mutex := m.getOrCreateMutex(id)
	mutex.Lock()
	defer mutex.Unlock()

	if _, err := m.items.read(id); err != nil {
		return fmt.Errorf("item with ID %s does not exist", id.String())
	}

	path := m.itemPath(id)
	if err := os.Remove(path); err != nil {
		return err
	}

	m.items.delete(id)
	m.deleteMutex(id) // Clean up the mutex after deleting the item
	return nil
}

// Count an item from the collection by its ID.
func (m *Manager[T]) Count() int {
	return m.items.count()
}
