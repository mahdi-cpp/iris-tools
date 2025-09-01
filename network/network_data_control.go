package network

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// https://chat.deepseek.com/a/chat/s/9b010f32-b23d-4f9b-ae0c-31a9b2c9408c

type Control[T any] struct {
	baseURL    string
	httpClient *http.Client
	mutex      sync.RWMutex
}

func NewNetworkManager[T any](baseURL string) *Control[T] {
	return &Control[T]{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (control *Control[T]) Read(endpoint string, requestBody interface{}) (*T, error) {
	control.mutex.RLock()
	defer control.mutex.RUnlock()

	//startTime := time.Now()

	// Prepare request body
	var body io.Reader
	if requestBody != nil {
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create POST request
	fullURL := control.baseURL + endpoint
	req, err := http.NewRequest(http.MethodPost, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("userID", "1")

	// Execute request
	resp, err := control.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read complete response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Consider both 200 (OK) and 201 (Created) as success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse error response
		var errorResp struct {
			Error   string `json:"error"`
			Details string `json:"details"`
		}
		if json.Unmarshal(responseBody, &errorResp) == nil {
			if errorResp.Details != "" {
				return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Details)
			}
			return nil, errors.New(errorResp.Error)
		}
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	data := new(T)
	if err := json.Unmarshal(responseBody, data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (response: %s)", err, string(responseBody))
	}

	//Log performance
	//duration := time.Since(startTime)
	//log.Printf("Search:  (in %v)", duration)

	return data, nil
}
