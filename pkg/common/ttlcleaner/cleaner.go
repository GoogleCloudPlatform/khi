// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ttlcleaner

import (
	"log/slog"
	"sync"
	"time"
)

// ExpirableTarget defines the interface for resource pools whose entries have TTLs and can be evicted.
type ExpirableTarget[K comparable] interface {
	// Expirations returns a snapshot mapping active keys to their expiration timestamps.
	Expirations() map[K]time.Time

	// Evict removes and cleans up the resource associated with the given key.
	Evict(key K) error
}

// TTLCleaner periodically sweeps an ExpirableTarget and evicts entries whose expiration timestamp has passed.
type TTLCleaner[K comparable] struct {
	target   ExpirableTarget[K]
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewTTLCleaner creates a new TTLCleaner instance for the given target and sweep interval.
func NewTTLCleaner[K comparable](target ExpirableTarget[K], interval time.Duration) *TTLCleaner[K] {
	return &TTLCleaner[K]{
		target:   target,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start launches the background goroutine that periodically performs cleanup.
func (c *TTLCleaner[K]) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChan:
				return
			case now := <-ticker.C:
				c.Sweep(now)
			}
		}
	}()
}

// Stop stops the background cleanup goroutine and waits for it to exit.
func (c *TTLCleaner[K]) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stopChan)
	c.mu.Unlock()

	c.wg.Wait()
}

// Sweep checks all active entries in the target, evicts those expired before now, and returns evicted keys.
func (c *TTLCleaner[K]) Sweep(now time.Time) []K {
	expirations := c.target.Expirations()
	var evictedKeys []K
	for key, expiresAt := range expirations {
		if now.After(expiresAt) {
			if err := c.target.Evict(key); err != nil {
				slog.Warn("Failed to evict expired entry in TTLCleaner", "error", err)
				continue
			}
			evictedKeys = append(evictedKeys, key)
		}
	}
	return evictedKeys
}
