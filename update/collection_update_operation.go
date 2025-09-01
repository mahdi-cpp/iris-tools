package update

import "reflect"

// CollectionUpdateOp defines operations for slice fields
//type CollectionUpdateOp[T comparable] struct {
//	FullReplace *[]T // Pointer to new slice (full replacement)
//	Add         []T  // Items to add
//	Remove      []T  // Items to remove
//}

// CollectionUpdateOp defines operations for slice fields
type CollectionUpdateOp[T any] struct {
	FullReplace *[]T // Full slice replacement
	Add         []T  // Items to add
	Remove      []T  // Items to remove (by ID or value)
}

// KeyExtractor defines a function to get ID from nested structs
type KeyExtractor[T any, K comparable] func(T) K

// ApplyCollectionUpdate handles slice updates with add/remove operations
func ApplyCollectionUpdate[T comparable](current []T, op CollectionUpdateOp[T]) []T {

	switch {
	case op.FullReplace != nil:
		return *op.FullReplace

	case len(op.Add) > 0 || len(op.Remove) > 0:
		// Create lookup set for existing items
		set := make(map[T]bool, len(current))
		for _, v := range current {
			set[v] = true
		}

		// Add new items
		for _, v := range op.Add {
			if !set[v] {
				current = append(current, v)
				set[v] = true
			}
		}

		// Remove items if specified
		if len(op.Remove) > 0 {
			removeSet := make(map[T]bool, len(op.Remove))
			for _, v := range op.Remove {
				removeSet[v] = true
			}

			newSlice := make([]T, 0, len(current))
			for _, v := range current {
				if !removeSet[v] {
					newSlice = append(newSlice, v)
				}
			}
			return newSlice
		}
	}
	return current
}

// UpdaterConfig holds update configuration
type UpdaterConfig[T any, U any] struct {
	ScalarUpdaters     []func(*T, U) // Field update functions
	CollectionUpdaters []func(*T, U) // Collection update functions
	NestedUpdaters     []func(*T, U) // For nested struct operations
	PostUpdateHooks    []func(*T)    // Finalization hooks
}

// NewUpdater creates a new updater instance
func NewUpdater[T any, U any]() *UpdaterConfig[T, U] {
	return &UpdaterConfig[T, U]{
		ScalarUpdaters:     make([]func(*T, U), 0),
		CollectionUpdaters: make([]func(*T, U), 0),
		PostUpdateHooks:    make([]func(*T), 0),
	}
}

func (u *UpdaterConfig[T, U]) AddScalarUpdater(fn func(*T, U)) {
	u.ScalarUpdaters = append(u.ScalarUpdaters, fn)
}

func (u *UpdaterConfig[T, U]) AddCollectionUpdater(fn func(*T, U)) {
	u.CollectionUpdaters = append(u.CollectionUpdaters, fn)
}

func (u *UpdaterConfig[T, U]) AddPostUpdateHook(fn func(*T)) {
	u.PostUpdateHooks = append(u.PostUpdateHooks, fn)
}

// Apply executes all updates including nested operations
func (u *UpdaterConfig[T, U]) Apply(target *T, update U) {

	// Apply scalar updates
	for _, fn := range u.ScalarUpdaters {
		fn(target, update)
	}

	// Apply collection updates
	for _, fn := range u.CollectionUpdaters {
		fn(target, update)
	}

	// Apply nested updates
	for _, fn := range u.NestedUpdaters {
		fn(target, update)
	}

	// Run post-update hooks
	for _, fn := range u.PostUpdateHooks {
		fn(target)
	}
}

// ---------------------

// AddNestedUpdater registers nested struct handlers
func (u *UpdaterConfig[T, U]) AddNestedUpdater(fn func(*T, U)) {
	u.NestedUpdaters = append(u.NestedUpdaters, fn)
}

// updateField uses reflection to update struct fields
func updateField[T any](item *T, field string, value interface{}) {
	structValue := reflect.ValueOf(item).Elem()
	fieldValue := structValue.FieldByName(field)

	if fieldValue.IsValid() && fieldValue.CanSet() {
		val := reflect.ValueOf(value)
		if val.Type().AssignableTo(fieldValue.Type()) {
			fieldValue.Set(val)
		}
	}
}

// ApplyCollectionUpdateByID handles nested struct updates using ID comparison
func ApplyCollectionUpdateByID[T any, K comparable](
	current []T,
	op CollectionUpdateOp[T],
	keyExtractor KeyExtractor[T, K],
) []T {
	switch {
	case op.FullReplace != nil:
		return *op.FullReplace

	case len(op.Add) > 0 || len(op.Remove) > 0:
		// Create ID->value map for existing items
		currentMap := make(map[K]T)
		for _, item := range current {
			key := keyExtractor(item)
			currentMap[key] = item
		}

		// Add new items
		for _, item := range op.Add {
			key := keyExtractor(item)
			currentMap[key] = item
		}

		// Remove specified items
		for _, item := range op.Remove {
			key := keyExtractor(item)
			delete(currentMap, key)
		}

		// Convert map back to slice
		newSlice := make([]T, 0, len(currentMap))
		for _, item := range currentMap {
			newSlice = append(newSlice, item)
		}
		return newSlice

	default:
		return current
	}
}

// NestedFieldUpdate defines a nested struct update operation
type NestedFieldUpdate[T any] struct {
	ID    int
	Field string
	Value interface{}
}

// ApplyNestedUpdate updates specific fields in nested structs
func ApplyNestedUpdate[T any](
	slice []T,
	updates []NestedFieldUpdate[T],
	keyExtractor KeyExtractor[T, int],
) []T {
	// Create ID->index map
	idToIndex := make(map[int]int)
	for i, item := range slice {
		id := keyExtractor(item)
		idToIndex[id] = i
	}

	for _, update := range updates {
		if idx, exists := idToIndex[update.ID]; exists {
			item := &slice[idx]
			updateField(item, update.Field, update.Value)
		}
	}
	return slice
}
