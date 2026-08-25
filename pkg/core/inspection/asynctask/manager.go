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

package asynctask

import (
	"context"
	"sync"
)

// TaskStatus represents the lifecycle state of an asynchronous background task.
type TaskStatus int

const (
	// StatusPending indicates the task is queued but has not yet started.
	StatusPending TaskStatus = iota

	// StatusRunning indicates the task is currently executing in the background.
	StatusRunning

	// StatusCompleted indicates the task has finished successfully with a result.
	StatusCompleted

	// StatusFailed indicates the task execution encountered an error.
	StatusFailed
)

// AsyncTaskResult contains the current execution status, value, or error of an asynchronous task.
type AsyncTaskResult[T any] struct {
	Status TaskStatus
	Value  T
	Error  error
}

type taskFlight[K comparable, T any] struct {
	inputKey K
	cancel   context.CancelFunc
}

// AsyncTaskManager coordinates background tasks for inspection task slots.
// It provides memoization of results by input key, non-blocking execution status checks,
// and automatic cancellation of outdated in-flight tasks when input keys change.
type AsyncTaskManager[K comparable, T any] struct {
	mu       sync.Mutex
	cache    map[string]map[K]AsyncTaskResult[T]
	inFlight map[string]*taskFlight[K, T]
}

// NewAsyncTaskManager creates a new AsyncTaskManager instance.
func NewAsyncTaskManager[K comparable, T any]() *AsyncTaskManager[K, T] {
	return &AsyncTaskManager[K, T]{
		cache:    make(map[string]map[K]AsyncTaskResult[T]),
		inFlight: make(map[string]*taskFlight[K, T]),
	}
}

// DoAsyncOrGet queries the current execution status or cached result for the given slotKey and inputKey.
// If the result is already cached, it returns the cached result immediately.
// If a task with the same inputKey is already running, it returns StatusRunning immediately without blocking.
// If a task with a different inputKey was running for this slotKey, it cancels the previous task and starts a new one.
// If no task has started, it initiates the background worker and returns StatusRunning immediately.
func (m *AsyncTaskManager[K, T]) DoAsyncOrGet(
	slotKey string,
	inputKey K,
	worker func(ctx context.Context) (T, error),
) AsyncTaskResult[T] {
	m.mu.Lock()

	// 1. Check cache for slotKey and inputKey
	if slotCache, ok := m.cache[slotKey]; ok {
		if res, exists := slotCache[inputKey]; exists {
			m.mu.Unlock()
			return res
		}
	}

	// 2. Check in-flight task for slotKey
	if flight, exists := m.inFlight[slotKey]; exists {
		if flight.inputKey == inputKey {
			m.mu.Unlock()
			return AsyncTaskResult[T]{Status: StatusRunning}
		}
		// Input key changed: cancel outdated in-flight task
		flight.cancel()
		delete(m.inFlight, slotKey)
	}

	// 3. Start a new background task
	callCtx, callCancel := context.WithCancel(context.Background())
	flight := &taskFlight[K, T]{
		inputKey: inputKey,
		cancel:   callCancel,
	}
	m.inFlight[slotKey] = flight
	m.mu.Unlock()

	go func() {
		val, err := worker(callCtx)

		m.mu.Lock()
		defer m.mu.Unlock()

		// Verify this flight is still the active one for this slotKey and inputKey
		current, exists := m.inFlight[slotKey]
		if !exists || current != flight {
			return
		}

		delete(m.inFlight, slotKey)

		// Do not cache if the worker was canceled
		if callCtx.Err() != nil {
			return
		}

		if m.cache[slotKey] == nil {
			m.cache[slotKey] = make(map[K]AsyncTaskResult[T])
		}

		if err != nil {
			m.cache[slotKey][inputKey] = AsyncTaskResult[T]{
				Status: StatusFailed,
				Error:  err,
			}
		} else {
			m.cache[slotKey][inputKey] = AsyncTaskResult[T]{
				Status: StatusCompleted,
				Value:  val,
			}
		}
	}()

	return AsyncTaskResult[T]{Status: StatusRunning}
}

// Cancel cancels the active in-flight task for slotKey, if one exists.
func (m *AsyncTaskManager[K, T]) Cancel(slotKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flight, exists := m.inFlight[slotKey]; exists {
		flight.cancel()
		delete(m.inFlight, slotKey)
	}
}

// Reset cancels any in-flight task and clears all cached results for slotKey.
func (m *AsyncTaskManager[K, T]) Reset(slotKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if flight, exists := m.inFlight[slotKey]; exists {
		flight.cancel()
		delete(m.inFlight, slotKey)
	}
	delete(m.cache, slotKey)
}

// Close cancels all in-flight tasks and clears all caches across all slots.
func (m *AsyncTaskManager[K, T]) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, flight := range m.inFlight {
		flight.cancel()
	}
	m.inFlight = make(map[string]*taskFlight[K, T])
	m.cache = make(map[string]map[K]AsyncTaskResult[T])
}
