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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/google/go-cmp/cmp"
)

type mockMetricLogCountFetcher struct {
	delay    time.Duration
	count    int64
	err      error
	callHook func(ctx context.Context)
}

func (m *mockMetricLogCountFetcher) QueryMetricCount(ctx context.Context, container googlecloud.ResourceContainer, metricFilter string, startTime, endTime time.Time) (int64, error) {
	if m.callHook != nil {
		m.callHook(ctx)
	}
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.count, m.err
}

func TestCachedStructuredLogEstimator_CacheHit(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()
	defer cachedEstimator.Close()

	var providerCalls int64
	provider := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		atomic.AddInt64(&providerCalls, 1)
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{count: 100}, nil, nil), nil
	}

	query := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       []LoggingMonitoringMatcher{ResourceLabel("cluster_name", Exact("cluster-1"))},
	}
	container := googlecloud.Project("test-project")
	now := time.Now()
	startTime := now.Add(-time.Hour)
	endTime := now

	// First call -> executes provider
	res1, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-1", container, query, startTime, endTime, provider)
	if err != nil {
		t.Fatalf("first EstimateWithTaskSlot() error = %v", err)
	}
	if res1.EstimatedCount != 100 {
		t.Errorf("res1.EstimatedCount = %d, want 100", res1.EstimatedCount)
	}

	// Second call with identical inputs -> cache hit, provider is not called again
	res2, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-1", container, query, startTime, endTime, provider)
	if err != nil {
		t.Fatalf("second EstimateWithTaskSlot() error = %v", err)
	}
	if diff := cmp.Diff(res1, res2); diff != "" {
		t.Errorf("cached result mismatch (-want +got):\n%s", diff)
	}

	if calls := atomic.LoadInt64(&providerCalls); calls != 1 {
		t.Errorf("providerCalls = %d, want 1", calls)
	}
}

func TestCachedStructuredLogEstimator_SingleflightDeduplication(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()
	defer cachedEstimator.Close()

	var providerCalls int64
	provider := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		atomic.AddInt64(&providerCalls, 1)
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{
			delay: 50 * time.Millisecond,
			count: 200,
		}, nil, nil), nil
	}

	query := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       []LoggingMonitoringMatcher{ResourceLabel("cluster_name", Exact("cluster-1"))},
	}
	container := googlecloud.Project("test-project")
	now := time.Now()
	startTime := now.Add(-time.Hour)
	endTime := now

	var wg sync.WaitGroup
	results := make([]*EstimateResult, 5)
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-slot", container, query, startTime, endTime, provider)
			results[idx] = res
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < 5; i++ {
		if errs[i] != nil {
			t.Errorf("results[%d] had error: %v", i, errs[i])
		}
		if results[i] == nil || results[i].EstimatedCount != 200 {
			t.Errorf("results[%d] = %v, want EstimatedCount=200", i, results[i])
		}
	}

	if calls := atomic.LoadInt64(&providerCalls); calls != 1 {
		t.Errorf("providerCalls = %d, want 1", calls)
	}
}

func TestCachedStructuredLogEstimator_OutdatedQueryCancellation(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()
	defer cachedEstimator.Close()

	query1Cancelled := make(chan struct{})
	provider1 := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{
			delay: 500 * time.Millisecond,
			callHook: func(queryCtx context.Context) {
				go func() {
					<-queryCtx.Done()
					close(query1Cancelled)
				}()
			},
			count: 100,
		}, nil, nil), nil
	}

	provider2 := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{
			delay: 10 * time.Millisecond,
			count: 300,
		}, nil, nil), nil
	}

	query1 := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       []LoggingMonitoringMatcher{ResourceLabel("cluster_name", Exact("cluster-1"))},
	}
	query2 := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       []LoggingMonitoringMatcher{ResourceLabel("cluster_name", Exact("cluster-2"))},
	}

	container := googlecloud.Project("test-project")
	now := time.Now()
	startTime := now.Add(-time.Hour)
	endTime := now

	// Start query1 on taskSlot
	go func() {
		_, _ = cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-slot-A", container, query1, startTime, endTime, provider1)
	}()

	// Allow query1 to start
	time.Sleep(20 * time.Millisecond)

	// Now send query2 on the SAME taskSlot -> should cancel query1
	res2, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-slot-A", container, query2, startTime, endTime, provider2)
	if err != nil {
		t.Fatalf("query2 EstimateWithTaskSlot() error = %v", err)
	}
	if res2.EstimatedCount != 300 {
		t.Errorf("res2.EstimatedCount = %d, want 300", res2.EstimatedCount)
	}

	// Verify query1's context was cancelled
	select {
	case <-query1Cancelled:
		// Success: query1 was cancelled
	case <-time.After(200 * time.Millisecond):
		t.Errorf("expected query1 to be cancelled when query2 was sent to the same task slot")
	}
}

func TestCachedStructuredLogEstimator_IdenticalQueryDoesNotCancel(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()
	defer cachedEstimator.Close()

	var providerCalls int64
	provider := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		atomic.AddInt64(&providerCalls, 1)
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{
			delay: 50 * time.Millisecond,
			count: 150,
		}, nil, nil), nil
	}

	query := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       []LoggingMonitoringMatcher{ResourceLabel("cluster_name", Exact("cluster-1"))},
	}
	container := googlecloud.Project("test-project")
	now := time.Now()
	startTime := now.Add(-time.Hour)
	endTime := now

	var wg sync.WaitGroup
	var res1, res2 *EstimateResult
	var err1, err2 error

	wg.Add(2)

	go func() {
		defer wg.Done()
		res1, err1 = cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-slot-B", container, query, startTime, endTime, provider)
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		// Same query on same slot -> joins flight, should NOT cancel
		res2, err2 = cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-slot-B", container, query, startTime, endTime, provider)
	}()

	wg.Wait()

	if err1 != nil {
		t.Errorf("call 1 returned unexpected error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("call 2 returned unexpected error: %v", err2)
	}
	if res1 == nil || res1.EstimatedCount != 150 {
		t.Errorf("res1 = %v, want EstimatedCount=150", res1)
	}
	if res2 == nil || res2.EstimatedCount != 150 {
		t.Errorf("res2 = %v, want EstimatedCount=150", res2)
	}
	if calls := atomic.LoadInt64(&providerCalls); calls != 1 {
		t.Errorf("providerCalls = %d, want 1", calls)
	}
}

func TestCachedStructuredLogEstimator_ErrorNotCached(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()
	defer cachedEstimator.Close()

	var attempts int64
	provider := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		current := atomic.AddInt64(&attempts, 1)
		if current == 1 {
			return nil, errors.New("network failure")
		}
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{count: 500}, nil, nil), nil
	}

	query := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
	}
	container := googlecloud.Project("test-project")
	now := time.Now()

	// Attempt 1 fails
	_, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-err", container, query, now.Add(-time.Hour), now, provider)
	if err == nil {
		t.Fatalf("first attempt expected error, got nil")
	}

	// Attempt 2 succeeds (retry is allowed)
	res, err := cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-err", container, query, now.Add(-time.Hour), now, provider)
	if err != nil {
		t.Fatalf("second attempt error = %v", err)
	}
	if res.EstimatedCount != 500 {
		t.Errorf("res.EstimatedCount = %d, want 500", res.EstimatedCount)
	}
}

func TestCachedStructuredLogEstimator_Close(t *testing.T) {
	cachedEstimator := NewCachedStructuredLogEstimator()

	cancelled := make(chan struct{})
	provider := func(ctx context.Context, container googlecloud.ResourceContainer) (*StructuredLogEstimator, error) {
		return NewStructuredLogEstimator(&mockMetricLogCountFetcher{
			delay: 500 * time.Millisecond,
			callHook: func(queryCtx context.Context) {
				go func() {
					<-queryCtx.Done()
					close(cancelled)
				}()
			},
			count: 100,
		}, nil, nil), nil
	}

	query := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
	}
	container := googlecloud.Project("test-project")
	now := time.Now()

	go func() {
		_, _ = cachedEstimator.EstimateWithTaskSlot(context.Background(), "task-close", container, query, now.Add(-time.Hour), now, provider)
	}()

	time.Sleep(20 * time.Millisecond)

	// Closing estimator should cancel in-flight tasks and clear cache
	cachedEstimator.Close()

	select {
	case <-cancelled:
		// Successfully cancelled
	case <-time.After(200 * time.Millisecond):
		t.Errorf("expected in-flight task to be cancelled on Close()")
	}
}
