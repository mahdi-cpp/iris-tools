package image_loader

//https://chat.deepseek.com/a/chat/s/5f0242a7-4635-4f65-9841-5e2a1d67920c

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// ImageLoader handles loading and processing images
type ImageLoader struct {
	cache         *lru.Cache[string, *imageItem]
	localBasePath string
	httpClient    *http.Client
	generateLock  sync.Mutex
	metrics       LoaderMetrics
}

// New struct to store image data with access time
type imageItem struct {
	data       []byte
	lastAccess int64 // UnixNano timestamp
	size       int32 // Only for metrics tracking
}

type LoaderMetrics struct {
	mu                sync.Mutex
	FileLoads         int
	NetworkLoads      int
	Generations       int
	LoadErrors        int
	ExpiredItems      int
	EvictedItems      int
	CurrentCacheBytes int32 // Current memory used by cache
	TotalOriginal     int64 // Total original bytes processed
	TotalFinal        int64 // Total final bytes processed
	LoadDurations     []time.Duration
}

// GetLocalBasePath safely exposes the local base path
func (il *ImageLoader) GetLocalBasePath() string {
	return il.localBasePath
}

func NewImageLoader(count int, localBasePath string, expiration time.Duration) *ImageLoader {

	cache, _ := lru.New[string, *imageItem](count)

	loader := &ImageLoader{
		cache:         cache,
		localBasePath: localBasePath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 10,
			},
		},
	}

	// Start background cleaner (runs every expiration minute, expires after expiration minutes)
	if expiration > 0 {
		go loader.StartCleaner(time.Minute, expiration)
	}

	return loader
}

// LoadImage handles all image requests (main entry point)
func (il *ImageLoader) LoadImage(ctx context.Context, imageID string) ([]byte, error) {

	start := time.Now()

	if item, ok := il.cache.Get(imageID); ok {
		//fmt.Println("load of----------")
		return item.data, nil
	}

	// Determine source and load
	var data []byte
	var err error

	switch {
	case strings.HasPrefix(imageID, "http://") || strings.HasPrefix(imageID, "https://"):
		data, err = il.loadNetworkImage(ctx, imageID)
	case strings.HasPrefix(imageID, "gen:"):
		//data, err = il.generateImage(ctx, imageID)
	case strings.HasPrefix(imageID, "placeholder:"):
		//data, err = il.createPlaceholder(ctx, imageID)
	default:
		data, err = il.loadLocalImage(imageID)
	}

	// Track metrics
	duration := time.Since(start)
	fmt.Println(duration)

	il.metrics.mu.Lock()
	if err != nil {
		il.metrics.LoadErrors++
	} else {
		// After loading image, add to cache
		newItem := &imageItem{
			data:       data,
			lastAccess: time.Now().UnixNano(),
			size:       int32(len(data)),
		}
		il.cache.Add(imageID, newItem)

		il.metrics.LoadDurations = append(il.metrics.LoadDurations, duration)
		il.metrics.CurrentCacheBytes += newItem.size
	}
	il.metrics.mu.Unlock()

	return data, err
}

// StartCleaner background cleaner routine
func (il *ImageLoader) StartCleaner(interval, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		il.cleanExpired(ttl)
	}
}

func (il *ImageLoader) cleanExpired(ttl time.Duration) {
	keys := il.cache.Keys()
	now := time.Now()

	for _, key := range keys {
		if item, ok := il.cache.Peek(key); ok {

			if now.Sub(time.Unix(0, item.lastAccess)) > ttl {

				// Get size before removal
				sizeBefore := atomic.LoadInt32(&il.metrics.CurrentCacheBytes)
				//finalSize := int64(len(item.data))

				il.cache.Remove(key)

				il.metrics.mu.Lock()
				il.metrics.ExpiredItems++
				il.metrics.CurrentCacheBytes -= item.size
				sizeAfter := il.metrics.CurrentCacheBytes
				il.metrics.mu.Unlock()

				fmt.Printf("Expired item removed: %s\n", key)
				fmt.Printf("Size: %s\n", FormatBytes(item.size))
				fmt.Printf("Cache memory: %s → %s\n", FormatBytes(sizeBefore), FormatBytes(sizeAfter))
			}
		}
	}
}

func (il *ImageLoader) loadLocalImage(imageID string) ([]byte, error) {

	path := filepath.Join(il.localBasePath, imageID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Get original size BEFORE conversion
	originalSize := len(data)
	var result []byte

	// Validate image format
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid image format for %s: %w", imageID, err)
	}

	// Convert to standard format if needed
	switch strings.ToLower(format) {
	case "jpeg", "jpg", "png":
		result = data
	default:
		result, err = il.convertImageFormat(data, "jpeg")
		if err != nil {
			return nil, err
		}
	}

	// Get final size AFTER conversion
	finalSize := len(result)

	// Update metrics
	il.metrics.mu.Lock()
	il.metrics.FileLoads++
	il.metrics.TotalOriginal += int64(originalSize)
	il.metrics.TotalFinal += int64(finalSize)
	il.metrics.mu.Unlock()

	return result, nil
}

// loadNetworkImage downloads from URL
func (il *ImageLoader) loadNetworkImage(ctx context.Context, url string) ([]byte, error) {

	il.metrics.mu.Lock()
	il.metrics.NetworkLoads++
	il.metrics.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	resp, err := il.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	// Validate image
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("invalid image from %s: %w", url, err)
	}

	return data, nil
}

// convertImageFormat converts between image formats
func (il *ImageLoader) convertImageFormat(data []byte, targetFormat string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	switch strings.ToLower(targetFormat) {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, img)
	default:
		return nil, fmt.Errorf("unsupported format: %s", targetFormat)
	}

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (il *ImageLoader) Metrics() *LoaderMetrics {

	il.metrics.mu.Lock()
	defer il.metrics.mu.Unlock()

	// Calculate compression ratio
	if il.metrics.CurrentCacheBytes > 0 {
		//compressionRatio = float64(il.metrics.localBytesFinal) / float64(il.metrics.localBytesOriginal)
	}

	// Calculate average duration
	totalDuration := time.Duration(0)
	for _, d := range il.metrics.LoadDurations {
		totalDuration += d
	}

	//if count := len(il.metrics.LoadDurations); count > 0 {
	//	avgDuration = totalDuration / time.Duration(count)
	//}

	return &il.metrics
}
