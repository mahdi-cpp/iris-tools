package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Control[T any] struct {
	filePath string
	mutex    sync.RWMutex
}

func NewMetadataControl[T any](filePath string) *Control[T] {
	return &Control[T]{
		filePath: filePath,
	}
}

func (control *Control[T]) Read(requireExist bool) (*T, error) {
	control.mutex.RLock()
	defer control.mutex.RUnlock()

	data := new(T)
	file, err := os.ReadFile(control.filePath)
	if err != nil {
		if os.IsNotExist(err) && requireExist {
			return nil, fmt.Errorf("file %s does not exist", control.filePath)
		}
		if os.IsNotExist(err) { // If not requiring existence, return empty data
			return data, nil
		}
		return nil, err
	}

	if len(file) == 0 {
		return data, nil
	}

	if err := json.Unmarshal(file, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (control *Control[T]) Update(updateFunc func(*T) error) error {
	control.mutex.Lock()
	defer control.mutex.Unlock()

	// Read current data into a pointer
	data, err := control.readData()
	if err != nil {
		return err
	}

	// Apply updates to the pointer
	if err := updateFunc(data); err != nil {
		return err
	}

	// Write updated data
	return control.writeData(data)
}

func (control *Control[T]) readData() (*T, error) {
	data := new(T)
	file, err := os.ReadFile(control.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, err
	}

	if len(file) > 0 {
		if err := json.Unmarshal(file, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (control *Control[T]) writeData(data *T) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tempFile := control.filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return err
	}

	return os.Rename(tempFile, control.filePath)
}

func (control *Control[T]) Write(data *T) error {
	control.mutex.Lock()
	defer control.mutex.Unlock()
	return control.writeData(data)
}
