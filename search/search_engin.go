package search

import (
	"sort"
	"strings"
	"time"
)

// Criteria defines a filter function for type T
type Criteria[T any] func(T) bool

// IndexedItem holds original index and value of matched items
type IndexedItem[T any] struct {
	Index int
	Value T
}

// Search  returns filtered items with their original indices
func Search[T any](slice []T, criteria Criteria[T]) []IndexedItem[T] {
	var results []IndexedItem[T]
	for i, item := range slice {
		if criteria(item) {
			results = append(results, IndexedItem[T]{Index: i, Value: item})
		}
	}
	return results
}

// LessFunction defines a comparison function for sorting
type LessFunction[T any] func(a, b T) bool

// SortIndexedItems sorts search results by value
func SortIndexedItems[T any](items []IndexedItem[T], lessFn LessFunction[T]) {
	sort.Slice(items, func(i, j int) bool {
		return lessFn(items[i].Value, items[j].Value)
	})
}

// Common Filter Helpers (optional but useful)
// ---------------------------------------------------------------------

// StringContains checks if field contains query (case-insensitive)
func StringContains(field, query string) bool {
	return strings.Contains(strings.ToLower(field), strings.ToLower(query))
}

// TimeInRange checks if time is within [start, end]
func TimeInRange(t, start, end time.Time) bool {
	if !start.IsZero() && t.Before(start) {
		return false
	}
	if !end.IsZero() && t.After(end) {
		return false
	}
	return true
}

// IntInSlice checks if value exists in int slice
func IntInSlice(value int, slice []int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// StringInSlice checks if value exists in string slice
func StringInSlice(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
