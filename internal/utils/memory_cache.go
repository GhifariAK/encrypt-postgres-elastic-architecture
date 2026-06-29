package utils

import (
	"fmt"
	"sync"
	"time"
)

// CacheItem menyimpan daftar ID dan waktu kedaluwarsa
type CacheItem struct {
	KaryawanIDs []int
	ExpiredAt   time.Time
}

// MemoryCache adalah brankas thread-safe kita
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

// Global variable yang bisa diakses dari Handler
var RequestCache = &MemoryCache{
	items: make(map[string]CacheItem),
}

// Set menyimpan request_id ke memori selama durasi tertentu (misal: 5 menit)
func (c *MemoryCache) Set(reqID string, ids []int, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[reqID] = CacheItem{
		KaryawanIDs: ids,
		ExpiredAt:   time.Now().Add(duration),
	}
}

// Get mengambil daftar ID berdasarkan request_id
func (c *MemoryCache) Get(reqID string) ([]int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[reqID]
	if !exists {
		return nil, false
	}

	// Cek apakah sudah hangus
	if time.Now().After(item.ExpiredAt) {
		return nil, false // Anggap tidak ada kalau sudah kedaluwarsa
	}

	return item.KaryawanIDs, true
}

// GenerateRequestID membuat string unik acak berbasis waktu
func GenerateRequestID() string {
	return fmt.Sprintf("REQ-%d", time.Now().UnixNano())
}

// StartGarbageCollector adalah Goroutine yang berjalan selamanya di latar belakang
// untuk menghapus data yang sudah kedaluwarsa agar RAM tidak bocor (Memory Leak)
func (c *MemoryCache) StartGarbageCollector() {
	ticker := time.NewTicker(1 * time.Minute) // Cek setiap 1 menit
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for reqID, item := range c.items {
			if now.After(item.ExpiredAt) {
				delete(c.items, reqID) // Hapus data sampah
			}
		}
		c.mu.Unlock()
	}
}
