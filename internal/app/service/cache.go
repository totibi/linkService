package service

import (
	"linkService/internal/app/domain"
	"log/slog"
	"sync"
	"time"
)

type cacheEntry struct {
	link      *domain.Link
	expiresAt time.Time
}

// потокобезопасный кэш для оригинальных ссылок по короткому коду с максимальным размером и TTL
type linkCache struct {
	mu         sync.RWMutex
	items      map[string]*cacheEntry
	maxSize    int
	defaultTTL time.Duration
	stopCh     chan struct{}
}

func newLinkCache(maxSize int, defaultTTL time.Duration) *linkCache {
	c := &linkCache{
		items:      make(map[string]*cacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stopCh:     make(chan struct{}),
	}
	// Запускаем фоновую очистку раз в минуту
	go c.cleanupLoop(1 * time.Minute)
	return c
}

func (c *linkCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *linkCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for code, entry := range c.items {
		if now.After(entry.expiresAt) {
			delete(c.items, code)
			slog.Debug("Cache eviction (TTL)", "short_code", code)
		}
	}
}

// get возвращает запись из кэша (учитывает TTL)
func (c *linkCache) get(code string) (*domain.Link, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[code]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.link, true
}

// set добавляет запись с защитой от переполнения
func (c *linkCache) set(code string, link *domain.Link) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если достигнут лимит, удаляем самый старый по TTL
	if len(c.items) >= c.maxSize {
		var oldestCode string
		var oldestTime time.Time
		first := true
		for k, v := range c.items {
			if first || v.expiresAt.Before(oldestTime) {
				oldestCode = k
				oldestTime = v.expiresAt
				first = false
			}
		}
		if oldestCode != "" {
			delete(c.items, oldestCode)
			slog.Debug("Cache eviction (size limit)", "short_code", oldestCode)
		}
	}

	c.items[code] = &cacheEntry{
		link:      link,
		expiresAt: time.Now().Add(c.defaultTTL),
	}
	slog.Debug("Cache SET", "short_code", code)
}

// delete удаляет запись из кэша
func (c *linkCache) delete(code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, code)
	slog.Debug("Cache DELETE", "short_code", code)
}

// Stop останавливает фоновую очистку (для graceful shutdown)
func (c *linkCache) Stop() {
	close(c.stopCh)
	slog.Info("cache clearing stopped")
}
