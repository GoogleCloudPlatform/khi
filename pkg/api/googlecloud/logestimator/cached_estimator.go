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

package logestimator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
)

type flightCall struct {
	done   chan struct{}
	res    *EstimateResult
	err    error
	cancel context.CancelFunc
}

type taskFlight struct {
	cacheKey string
	cancel   context.CancelFunc
}

// EstimatorProvider is a factory function to construct a StructuredLogEstimator for a container.
type EstimatorProvider func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error)

// CachedStructuredLogEstimator manages cached log volume estimates, singleflight in-flight deduplication,
// and cancellation of outdated in-flight estimation queries per task slot.
type CachedStructuredLogEstimator struct {
	mu          sync.Mutex
	cache       map[string]*EstimateResult
	flights     map[string]*flightCall
	activeTasks map[string]*taskFlight
}

// NewCachedStructuredLogEstimator creates a new CachedStructuredLogEstimator instance.
func NewCachedStructuredLogEstimator() *CachedStructuredLogEstimator {
	return &CachedStructuredLogEstimator{
		cache:       make(map[string]*EstimateResult),
		flights:     make(map[string]*flightCall),
		activeTasks: make(map[string]*taskFlight),
	}
}

// EstimateWithTaskSlot estimates the log volume for the given query, returning cached results when available.
// If a query for the same input is already in flight, it deduplicates the request and waits for the shared result.
// If a previous in-flight estimation for taskSlotKey has a different query or time range, it cancels the previous request.
func (e *CachedStructuredLogEstimator) EstimateWithTaskSlot(
	ctx context.Context,
	taskSlotKey string,
	container googlecloud.ResourceContainer,
	query *StructuredLogQuery,
	startTime, endTime time.Time,
	provider EstimatorProvider,
) (*EstimateResult, error) {
	res, flight, pending, err := e.estimateWithTaskSlotInternal(ctx, taskSlotKey, container, query, startTime, endTime, provider)
	if !pending {
		return res, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-flight.done:
		return flight.res, flight.err
	}
}

// EstimateWithTaskSlotNonBlocking estimates the log volume for the given query asynchronously without blocking the caller.
// It returns (res, false, nil) if the result is already cached.
// It returns (nil, true, nil) if the estimation is currently in-flight (pending).
// It returns (nil, false, err) if an error occurred during estimation.
func (e *CachedStructuredLogEstimator) EstimateWithTaskSlotNonBlocking(
	ctx context.Context,
	taskSlotKey string,
	container googlecloud.ResourceContainer,
	query *StructuredLogQuery,
	startTime, endTime time.Time,
	provider EstimatorProvider,
) (*EstimateResult, bool, error) {
	res, _, pending, err := e.estimateWithTaskSlotInternal(ctx, taskSlotKey, container, query, startTime, endTime, provider)
	return res, pending, err
}

// estimateWithTaskSlotInternal handles the core estimation logic under lock and returns the active flightCall pointer.
func (e *CachedStructuredLogEstimator) estimateWithTaskSlotInternal(
	ctx context.Context,
	taskSlotKey string,
	container googlecloud.ResourceContainer,
	query *StructuredLogQuery,
	startTime, endTime time.Time,
	provider EstimatorProvider,
) (*EstimateResult, *flightCall, bool, error) {
	if query.Preset != EstimatedCountPresetNone {
		return &EstimateResult{
			MetricCount:       0,
			EstimatedCount:    0,
			CustomFilterRatio: 1.0,
			IsExact:           false,
			Preset:            query.Preset,
		}, nil, false, nil
	}

	cacheKey := fmt.Sprintf("%s|%s|%d|%d",
		container.Identifier(),
		query.GenerateCloudLoggingQuery(),
		startTime.UnixNano(),
		endTime.UnixNano(),
	)

	e.mu.Lock()

	// 1. Return cached result if present.
	if res, ok := e.cache[cacheKey]; ok {
		e.mu.Unlock()
		return res, nil, false, nil
	}

	// 2. If an in-flight query exists for this task slot with a different query, cancel it.
	if taskSlotKey != "" {
		if active, ok := e.activeTasks[taskSlotKey]; ok {
			if active.cacheKey != cacheKey {
				active.cancel()
				delete(e.activeTasks, taskSlotKey)
			}
		}
	}

	// 3. Join in-flight execution if one already exists for this cacheKey.
	if flight, ok := e.flights[cacheKey]; ok {
		if taskSlotKey != "" {
			e.activeTasks[taskSlotKey] = &taskFlight{
				cacheKey: cacheKey,
				cancel:   flight.cancel,
			}
		}
		e.mu.Unlock()
		return nil, flight, true, nil
	}

	// 4. Start a new in-flight execution.
	callCtx, callCancel := context.WithCancel(context.Background())
	flight := &flightCall{
		done:   make(chan struct{}),
		cancel: callCancel,
	}
	e.flights[cacheKey] = flight

	if taskSlotKey != "" {
		e.activeTasks[taskSlotKey] = &taskFlight{
			cacheKey: cacheKey,
			cancel:   callCancel,
		}
	}
	e.mu.Unlock()

	go func() {
		var res *EstimateResult
		var err error

		defer func() {
			e.mu.Lock()
			flight.res = res
			flight.err = err
			if err == nil && res != nil {
				e.cache[cacheKey] = res
			}
			delete(e.flights, cacheKey)
			if taskSlotKey != "" {
				if active, ok := e.activeTasks[taskSlotKey]; ok && active.cacheKey == cacheKey {
					delete(e.activeTasks, taskSlotKey)
				}
			}
			close(flight.done)
			e.mu.Unlock()
		}()

		estimator, providerErr := provider(callCtx, container)
		if providerErr != nil {
			err = providerErr
			return
		}
		res, err = estimator.Estimate(callCtx, container, query, startTime, endTime)
	}()

	return nil, flight, true, nil
}

// Close cancels all active in-flight requests and clears the estimation cache.
func (e *CachedStructuredLogEstimator) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, task := range e.activeTasks {
		task.cancel()
	}
	for _, flight := range e.flights {
		flight.cancel()
	}
	e.activeTasks = make(map[string]*taskFlight)
	e.flights = make(map[string]*flightCall)
	e.cache = make(map[string]*EstimateResult)
}
