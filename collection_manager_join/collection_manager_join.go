package collection_manager_join

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

const (
	recordStatusSize = 1
)

const (
	StatusActive  = 0x00
	StatusDeleted = 0x01
)

// JoinItem اینترفیسی برای آیتم‌های دارای کلید ترکیبی.
type JoinItem interface {
	GetCompositeKey() string
	GetRecordSize() int
}

// Timestampable یک اینترفیس برای آیتم‌هایی است که قابلیت زمان‌بندی دارند.
type Timestampable interface {
	SetCreatedAt(t time.Time)
	SetUpdatedAt(t time.Time)
}

type FileHandler struct {
	dataFile   *os.File
	mu         sync.RWMutex
	dirName    string
	recordSize int
}

func NewFileHandler(dirName string, fileName string, recordSize int) (*FileHandler, error) {
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating directory %s: %w", dirName, err)
	}

	dataFileName := filepath.Join(dirName, fileName+".db")

	dataFile, err := os.OpenFile(dataFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening data file: %w", err)
	}

	return &FileHandler{
		dataFile:   dataFile,
		dirName:    dirName,
		recordSize: recordSize,
	}, nil
}

func (h *FileHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.dataFile.Close()
}

func (h *FileHandler) WriteRecord(data []byte) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(data) > h.recordSize-recordStatusSize {
		return -1, fmt.Errorf("data size is larger than max record size (%d bytes)", h.recordSize-recordStatusSize)
	}

	offset, err := h.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return -1, fmt.Errorf("error seeking to end of data file: %w", err)
	}

	recordBuffer := make([]byte, h.recordSize)
	recordBuffer[0] = StatusActive
	copy(recordBuffer[recordStatusSize:], data)

	if _, err := h.dataFile.Write(recordBuffer); err != nil {
		return -1, fmt.Errorf("error writing record: %w", err)
	}

	return offset, nil
}

func (h *FileHandler) ReadRecord(offset int64) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: %d", offset)
	}

	recordBuffer := make([]byte, h.recordSize)
	n, err := h.dataFile.ReadAt(recordBuffer, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading block from data file at offset %d: %w", offset, err)
	}

	if n == 0 {
		return nil, fmt.Errorf("no data read at offset %d", offset)
	}

	if recordBuffer[0] == StatusDeleted {
		return nil, fmt.Errorf("record at offset %d is marked as deleted", offset)
	}

	dataLength := bytes.IndexByte(recordBuffer[recordStatusSize:], 0)
	if dataLength == -1 {
		dataLength = h.recordSize - recordStatusSize
	} else if dataLength == 0 {
		return nil, fmt.Errorf("empty data at offset %d", offset)
	}

	return recordBuffer[recordStatusSize : recordStatusSize+dataLength], nil
}

func (h *FileHandler) UpdateRecord(offset int64, data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(data) > h.recordSize-recordStatusSize {
		return fmt.Errorf("data size is larger than max record size (%d bytes)", h.recordSize-recordStatusSize)
	}

	recordBuffer := make([]byte, h.recordSize)
	recordBuffer[0] = StatusActive
	copy(recordBuffer[recordStatusSize:], data)

	if _, err := h.dataFile.WriteAt(recordBuffer, offset); err != nil {
		return fmt.Errorf("error updating record in data file: %w", err)
	}

	return nil
}

func (h *FileHandler) DeleteRecord(offset int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, err := h.dataFile.WriteAt([]byte{StatusDeleted}, offset); err != nil {
		return fmt.Errorf("error marking record as deleted: %w", err)
	}
	return nil
}

// Manager برای مدیریت آیتم‌های دارای کلید ترکیبی.
type Manager[T JoinItem] struct {
	fh          *FileHandler
	mu          sync.RWMutex
	dataCache   map[string]T   // کش اصلی برای کلید ترکیبی
	parentCache map[string][]T // کش جدید برای parentID
	closed      bool
}

func New[T JoinItem](dirName string, fileName string) (*Manager[T], error) {
	var dataItem T
	recordSize := dataItem.GetRecordSize()

	fh, err := NewFileHandler(dirName, fileName, recordSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create file handler: %w", err)
	}

	manager := &Manager[T]{
		fh:          fh,
		dataCache:   make(map[string]T),
		parentCache: make(map[string][]T),
	}

	if err := manager.loadAllDataToCache(); err != nil {
		return nil, fmt.Errorf("failed to load data to cache: %w", err)
	}

	return manager, nil
}

func (m *Manager[T]) loadAllDataToCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fileInfo, err := m.fh.dataFile.Stat()
	if err != nil {
		return fmt.Errorf("error getting data file info: %w", err)
	}
	fileSize := fileInfo.Size()

	for offset := int64(0); offset < fileSize; offset += int64(m.fh.recordSize) {
		data, err := m.fh.ReadRecord(offset)
		if err != nil {
			continue
		}

		var loadedItem T
		if err := json.Unmarshal(data, &loadedItem); err != nil {
			log.Printf("Error unmarshaling data at offset %d: %v", offset, err)
			continue
		}

		m.dataCache[loadedItem.GetCompositeKey()] = loadedItem

		// پر کردن کش جدید
		keyParts := strings.Split(loadedItem.GetCompositeKey(), ":")
		if len(keyParts) > 0 {
			parentID := keyParts[0]
			m.parentCache[parentID] = append(m.parentCache[parentID], loadedItem)
		}
	}
	log.Printf("Loaded %d items into cache from data.db", len(m.dataCache))
	return nil
}

func (m *Manager[T]) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	m.dataCache = nil
	m.parentCache = nil

	return m.fh.Close()
}

func (m *Manager[T]) Create(item T) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var zero T
	if m.closed {
		return zero, fmt.Errorf("manager is closed")
	}

	key := item.GetCompositeKey()
	if _, ok := m.dataCache[key]; ok {
		return zero, fmt.Errorf("item with key %s already exists", key)
	}

	// مدیریت زمان‌بندی در صورت وجود اینترفیس Timestampable
	if tsItem, ok := any(item).(Timestampable); ok {
		now := time.Now()
		tsItem.SetCreatedAt(now)
		tsItem.SetUpdatedAt(now)
	}

	data, err := json.Marshal(item)
	if err != nil {
		return zero, fmt.Errorf("error marshaling item: %w", err)
	}

	_, err = m.fh.WriteRecord(data)
	if err != nil {
		return zero, fmt.Errorf("error writing record to disk: %w", err)
	}

	m.dataCache[key] = item

	// به‌روزرسانی کش والد
	keyParts := strings.Split(key, ":")
	if len(keyParts) > 0 {
		parentID := keyParts[0]
		m.parentCache[parentID] = append(m.parentCache[parentID], item)
	}

	return item, nil
}

func (m *Manager[T]) Read(key string) (T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var zero T
	item, ok := m.dataCache[key]
	if !ok {
		return zero, fmt.Errorf("item not found with key: %s", key)
	}
	return item, nil
}

func (m *Manager[T]) ReadAll() ([]T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var items []T
	for _, item := range m.dataCache {
		items = append(items, item)
	}
	return items, nil
}

func (m *Manager[T]) Update(item T) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var zero T
	key := item.GetCompositeKey()
	if _, ok := m.dataCache[key]; !ok {
		return zero, fmt.Errorf("item with key %s does not exist", key)
	}

	// مدیریت زمان‌بندی
	if tsItem, ok := any(item).(Timestampable); ok {
		tsItem.SetUpdatedAt(time.Now())
	}

	offset, err := m.findRecordOffset(key)
	if err != nil {
		return zero, err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return zero, fmt.Errorf("error marshaling item: %w", err)
	}

	if err := m.fh.UpdateRecord(offset, data); err != nil {
		return zero, fmt.Errorf("error updating record on disk: %w", err)
	}

	m.dataCache[key] = item

	return item, nil
}

func (m *Manager[T]) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.dataCache[key]
	if !ok {
		return fmt.Errorf("item with key %s not found", key)
	}

	offset, err := m.findRecordOffset(key)
	if err != nil {
		return err
	}

	if err := m.fh.DeleteRecord(offset); err != nil {
		return err
	}

	delete(m.dataCache, key)

	// به‌روزرسانی کش والد
	keyParts := strings.Split(key, ":")
	if len(keyParts) > 0 {
		parentID := keyParts[0]
		items := m.parentCache[parentID]
		for i, v := range items {
			if v.GetCompositeKey() == key {
				m.parentCache[parentID] = append(items[:i], items[i+1:]...)
				break
			}
		}
	}

	return nil
}

func (m *Manager[T]) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.dataCache)
}

func (m *Manager[T]) findRecordOffset(key string) (int64, error) {
	fileInfo, err := m.fh.dataFile.Stat()
	if err != nil {
		return -1, fmt.Errorf("error getting data file info: %w", err)
	}
	fileSize := fileInfo.Size()

	for offset := int64(0); offset < fileSize; offset += int64(m.fh.recordSize) {
		data, err := m.fh.ReadRecord(offset)
		if err != nil {
			continue
		}

		var loadedItem T
		if err := json.Unmarshal(data, &loadedItem); err != nil {
			log.Printf("Error unmarshaling data at offset %d: %v", offset, err)
			continue
		}

		if loadedItem.GetCompositeKey() == key {
			return offset, nil
		}
	}

	return -1, fmt.Errorf("item with key %s not found", key)
}

// GetByParentID تمام آیتم‌های مربوط به یک کلید والد را برمی‌گرداند.
func (m *Manager[T]) GetByParentID(parentID uuid.UUID) ([]T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parentIDStr := parentID.String()

	items, ok := m.parentCache[parentIDStr]
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("no items found for parent ID: %s", parentIDStr)
	}

	return items, nil
}

func (m *Manager[T]) Clone(item T) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var zero T
	if m.closed {
		return zero, fmt.Errorf("manager is closed")
	}

	key := item.GetCompositeKey()
	if _, ok := m.dataCache[key]; ok {
		return zero, fmt.Errorf("item with key %s already exists", key)
	}

	data, err := json.Marshal(item)
	if err != nil {
		return zero, fmt.Errorf("error marshaling item: %w", err)
	}

	_, err = m.fh.WriteRecord(data)
	if err != nil {
		return zero, fmt.Errorf("error writing record to disk: %w", err)
	}

	m.dataCache[key] = item

	// به‌روزرسانی کش والد
	keyParts := strings.Split(key, ":")
	if len(keyParts) > 0 {
		parentID := keyParts[0]
		m.parentCache[parentID] = append(m.parentCache[parentID], item)
	}

	return item, nil
}
