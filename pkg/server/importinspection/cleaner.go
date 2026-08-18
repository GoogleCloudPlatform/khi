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

package importinspection

import (
	"sync"
	"time"
)

const (
	// DefaultCleanupInterval is the frequency at which the cleaner checks for expired sessions.
	DefaultCleanupInterval = 1 * time.Minute
)

// ImportSessionCleaner periodically inspects active sessions and cancels those that have exceeded their TTL.
type ImportSessionCleaner struct {
	manager  *ImportSessionManager
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	stopped  bool
	mu       sync.Mutex
}

// NewImportSessionCleaner creates a new cleaner instance associated with the given session manager.
func NewImportSessionCleaner(manager *ImportSessionManager, interval time.Duration) *ImportSessionCleaner {
	return &ImportSessionCleaner{
		manager:  manager,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start initiates the background goroutine that periodically performs cleanup.
func (c *ImportSessionCleaner) Start() {
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
				c.Cleanup(now)
			}
		}
	}()
}

// Stop terminates the background cleanup goroutine and waits for it to complete.
func (c *ImportSessionCleaner) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	close(c.stopChan)
	c.mu.Unlock()

	c.wg.Wait()
}

// Cleanup checks all active sessions, determines if any have exceeded their TTL at the given time, and aborts them.
func (c *ImportSessionCleaner) Cleanup(now time.Time) []string {
	sessions := c.manager.GetActiveSessions()
	var cleanedTokens []string
	for _, s := range sessions {
		s.mu.Lock()
		isExpired := now.After(s.ExpiresAt)
		s.mu.Unlock()

		if isExpired {
			if err := c.manager.AbortSession(s.Token); err == nil {
				cleanedTokens = append(cleanedTokens, s.Token)
			}
		}
	}
	return cleanedTokens
}
