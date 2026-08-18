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

package workbench

import (
	"sync"
	"time"
)

// SweeperTarget defines the target operations required by Sweeper to inspect leases and evict expired sessions.
type SweeperTarget interface {
	Leases() map[string]time.Time
	Remove(workbenchID string)
}

// Sweeper periodically inspects workbench leases and requests removal of expired sessions.
type Sweeper struct {
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewSweeper creates a new Sweeper instance with the given execution interval.
func NewSweeper(interval time.Duration) *Sweeper {
	return &Sweeper{
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Run starts the periodic background sweep against the target manager.
func (s *Sweeper) Run(target SweeperTarget) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				return
			case t := <-ticker.C:
				s.Sweep(target, t)
			}
		}
	}()
}

// Sweep inspects target leases and requests removal of sessions that expired before now.
func (s *Sweeper) Sweep(target SweeperTarget, now time.Time) int {
	leases := target.Leases()
	evictedCount := 0
	for id, expiresAt := range leases {
		if now.After(expiresAt) {
			target.Remove(id)
			evictedCount++
		}
	}
	return evictedCount
}

// Stop terminates the sweeper goroutine and waits for completion.
func (s *Sweeper) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopChan)
	s.running = false
	s.mu.Unlock()

	s.wg.Wait()
}
