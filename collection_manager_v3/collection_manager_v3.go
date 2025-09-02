package collection_manager_v3

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mahdi-cpp/iris-tools/metadata"
	"github.com/mahdi-cpp/iris-tools/registery"
)

type CollectionItem interface {
	SetID(string)
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	GetID() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

type storage[T CollectionItem] interface {
	ReadAll(requireExist bool) ([]T, error)
	CreateItem(item T) error
	UpdateItem(item T) error
	DeleteItem(id string) error
}

type singleFileStorage[T CollectionItem] struct {
	ctrl *metadata.Control[[]T]
}

func (s *singleFileStorage[T]) ReadAll(requireExist bool) ([]T, error) {
	dataPtr, err := s.ctrl.Read(requireExist)
	if err != nil {
		return nil, err
	}
	if dataPtr == nil {
		return []T{}, nil
	}
	return *dataPtr, nil
}

func (s *singleFileStorage[T]) CreateItem(item T) error {
	items, err := s.ReadAll(false)
	if err != nil {
		return err
	}
	items = append(items, item)
	return s.ctrl.Write(&items)
}

func (s *singleFileStorage[T]) UpdateItem(updatedItem T) error {
	items, err := s.ReadAll(false)
	if err != nil {
		return err
	}
	found := false
	for i, item := range items {
		if item.GetID() == updatedItem.GetID() {
			items[i] = updatedItem
			found = true
			break
		}
	}
	if !found {
		return errors.New("item not found")
	}
	return s.ctrl.Write(&items)
}

func (s *singleFileStorage[T]) DeleteItem(id string) error {
	items, err := s.ReadAll(false)
	if err != nil {
		return err
	}
	var newItems []T
	for _, item := range items {
		if item.GetID() != id {
			newItems = append(newItems, item)
		}
	}
	return s.ctrl.Write(&newItems)
}

type directoryStorage[T CollectionItem] struct {
	baseDir string
}

func (d *directoryStorage[T]) itemPath(id string) string {
	return filepath.Join(d.baseDir, id+".json")
}

func (d *directoryStorage[T]) ReadAll(requireExist bool) ([]T, error) {
	if _, err := os.Stat(d.baseDir); err != nil {
		if os.IsNotExist(err) {
			if requireExist {
				return nil, err
			}
			return []T{}, nil
		}
		return nil, err
	}

	entries, err := os.ReadDir(d.baseDir)
	if err != nil {
		return nil, err
	}

	var items []T
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if filepath.Ext(filename) != ".json" {
			continue
		}

		id := strings.TrimSuffix(filename, filepath.Ext(filename))
		item, err := d.readItem(id)
		if err != nil {
			fmt.Println("collection_manager_v3:", err.Error())
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (d *directoryStorage[T]) readItem(id string) (T, error) {
	var zero T
	path := d.itemPath(id)
	ctrl := metadata.NewMetadataControl[T](path)

	dataPtr, err := ctrl.Read(true)
	if err != nil {
		return zero, err
	}
	if dataPtr == nil {
		return zero, errors.New("metadata not found")
	}
	return *dataPtr, nil
}

func (d *directoryStorage[T]) CreateItem(item T) error {
	if err := os.MkdirAll(d.baseDir, 0755); err != nil {
		return err
	}
	path := d.itemPath(item.GetID())
	ctrl := metadata.NewMetadataControl[T](path)
	return ctrl.Write(&item)
}

func (d *directoryStorage[T]) UpdateItem(item T) error {
	path := d.itemPath(item.GetID())
	ctrl := metadata.NewMetadataControl[T](path)
	return ctrl.Write(&item)
}

func (d *directoryStorage[T]) DeleteItem(id string) error {
	path := d.itemPath(id)
	return os.Remove(path)
}

type Manager[T CollectionItem] struct {
	storage storage[T]
	items   *registery.Registry[T]
}

type SortOptions struct {
	SortBy    string
	SortOrder string
}

func NewCollectionManager[T CollectionItem](path string, requireExist bool) (*Manager[T], error) {
	var store storage[T]

	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			store = &directoryStorage[T]{baseDir: path}
		} else {
			store = &singleFileStorage[T]{ctrl: metadata.NewMetadataControl[[]T](path)}
		}
	} else {
		if strings.HasSuffix(path, ".json") {
			store = &singleFileStorage[T]{ctrl: metadata.NewMetadataControl[[]T](path)}
		} else {
			store = &directoryStorage[T]{baseDir: path}
		}
	}

	manager := &Manager[T]{
		storage: store,
		items:   registery.NewRegistry[T](),
	}

	items, err := manager.storage.ReadAll(requireExist)
	if err != nil {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}

	for _, item := range items {
		manager.items.Register(item.GetID(), item)
	}

	return manager, nil
}

func (manager *Manager[T]) Create(newItem T) (T, error) {
	u7, err := uuid.NewV7()
	if err != nil {
		var zero T
		return zero, fmt.Errorf("error generating UUIDv7: %w", err)
	}

	newItem.SetID(u7.String())
	newItem.SetCreatedAt(time.Now())
	newItem.SetUpdatedAt(time.Now())

	if err := manager.storage.CreateItem(newItem); err != nil {
		return newItem, err
	}

	manager.items.Register(newItem.GetID(), newItem)
	return newItem, nil
}

func (manager *Manager[T]) Update(updatedItem T) (T, error) {
	updatedItem.SetUpdatedAt(time.Now())
	if err := manager.storage.UpdateItem(updatedItem); err != nil {
		return updatedItem, err
	}
	manager.items.Update(updatedItem.GetID(), updatedItem)
	return updatedItem, nil
}

func (manager *Manager[T]) Delete(id string) error {
	if err := manager.storage.DeleteItem(id); err != nil {
		return err
	}
	manager.items.Delete(id)
	return nil
}

func (manager *Manager[T]) Get(id string) (T, error) {
	return manager.items.Get(id)
}

func (manager *Manager[T]) GetList(filterFunc func(T) bool) ([]T, error) {
	allItems := manager.items.GetAllValues()
	var result []T
	for _, item := range allItems {
		if filterFunc == nil || filterFunc(item) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (manager *Manager[T]) GetAll() ([]T, error) {
	return manager.items.GetAllValues(), nil
}

func (manager *Manager[T]) GetBy(filterFunc func(T) bool) ([]T, error) {
	return manager.GetList(filterFunc)
}

func (manager *Manager[T]) SortItems(items []T, options SortOptions) []T {
	if options.SortBy == "" {
		return items
	}

	sort.Slice(items, func(i, j int) bool {
		a := items[i]
		b := items[j]

		switch options.SortBy {
		case "id":
			if options.SortOrder == "asc" {
				return a.GetID() < b.GetID()
			}
			return a.GetID() > b.GetID()
		case "creationDate":
			if options.SortOrder == "asc" {
				return a.GetCreatedAt().Before(b.GetCreatedAt())
			}
			return a.GetCreatedAt().After(b.GetCreatedAt())
		case "modificationDate":
			if options.SortOrder == "asc" {
				return a.GetUpdatedAt().Before(b.GetUpdatedAt())
			}
			return a.GetUpdatedAt().After(b.GetUpdatedAt())
		default:
			return false
		}
	})

	return items
}

func (manager *Manager[T]) GetSortedList(filterFunc func(T) bool, sortBy string, sortOrder string) ([]T, error) {
	items, err := manager.GetList(filterFunc)
	if err != nil {
		return nil, err
	}
	return manager.SortItems(items, SortOptions{
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}), nil
}

func (manager *Manager[T]) GetAllSorted(sortBy string, sortOrder string) ([]T, error) {
	items, err := manager.GetAll()
	if err != nil {
		return nil, err
	}
	return manager.SortItems(items, SortOptions{SortBy: sortBy, SortOrder: sortOrder}), nil
}
