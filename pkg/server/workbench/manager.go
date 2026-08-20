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
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"golang.org/x/sync/singleflight"
)

var (
	// ErrWorkbenchNotFound indicates that the requested workbench ID was not found or expired.
	ErrWorkbenchNotFound = errors.New("workbench session not found or expired")
	// ErrWorkbenchClosed indicates that the requested workbench session has been closed.
	ErrWorkbenchClosed = errors.New("workbench session has been closed")
)

var _ SweeperTarget = (*WorkbenchManager)(nil)

// WorkbenchManager coordinates the lifecycle and in-memory caching of Workbench sessions.
type WorkbenchManager struct {
	mu               sync.RWMutex
	workbenches      map[string]*Workbench
	leases           map[string]time.Time
	inspectionServer *coreinspection.InspectionTaskServer
	ttl              time.Duration
	sweeper          *Sweeper
	loadGroup        singleflight.Group
}

// NewWorkbenchManager creates a new WorkbenchManager instance with automatic background sweeping.
func NewWorkbenchManager(inspectionServer *coreinspection.InspectionTaskServer, ttl time.Duration, sweeperInterval time.Duration) *WorkbenchManager {
	mgr := &WorkbenchManager{
		workbenches:      make(map[string]*Workbench),
		leases:           make(map[string]time.Time),
		inspectionServer: inspectionServer,
		ttl:              ttl,
	}

	if sweeperInterval > 0 {
		mgr.sweeper = NewSweeper(sweeperInterval)
		mgr.sweeper.Run(mgr)
	}

	return mgr
}

// GetOrOpen retrieves an existing active Workbench session or loads the dataset into a new one.
func (m *WorkbenchManager) GetOrOpen(ctx context.Context, workbenchID string, inspectionID string, onProgress ProgressCallback) (*Workbench, error) {
	if onProgress == nil {
		onProgress = func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
			return nil
		}
	}

	// Check if already open and active for the same inspection dataset
	m.mu.Lock()
	wb, ok := m.workbenches[workbenchID]
	lease, hasLease := m.leases[workbenchID]
	if ok && hasLease && lease.After(time.Now()) && !wb.IsClosed() && wb.InspectionID() == inspectionID {
		m.leases[workbenchID] = time.Now().Add(m.ttl)
		m.mu.Unlock()
		if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_READY, 100, "Workbench attached."); err != nil {
			return nil, err
		}
		return wb, nil
	}
	if ok && wb != nil && !wb.IsClosed() {
		wb.Close()
		delete(m.workbenches, workbenchID)
		delete(m.leases, workbenchID)
	}
	m.mu.Unlock()

	// Deduplicate concurrent dataset loading for the same workbenchID using singleflight
	res, err, _ := m.loadGroup.Do(workbenchID, func() (any, error) {
		m.mu.Lock()
		if wb, ok := m.workbenches[workbenchID]; ok && !wb.IsClosed() && wb.InspectionID() == inspectionID {
			m.leases[workbenchID] = time.Now().Add(m.ttl)
			m.mu.Unlock()
			return wb, nil
		}
		m.mu.Unlock()

		if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INITIALIZING, 0, "Initializing workbench session..."); err != nil {
			return nil, err
		}

		reader, totalSize, err := m.loadInspectionData(inspectionID)
		if err != nil {
			return nil, fmt.Errorf("failed to load inspection data: %w", err)
		}
		defer reader.Close()

		if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_READING_FILE, 10, "Opening inspection dataset..."); err != nil {
			return nil, err
		}

		loadedWb, err := NewWorkbenchFromReader(ctx, workbenchID, inspectionID, reader, totalSize, onProgress)
		if err != nil {
			return nil, err
		}

		m.mu.Lock()
		if oldWb, exists := m.workbenches[workbenchID]; exists && oldWb != nil && oldWb != loadedWb {
			oldWb.Close()
		}
		m.workbenches[workbenchID] = loadedWb
		m.leases[workbenchID] = time.Now().Add(m.ttl)
		m.mu.Unlock()

		return loadedWb, nil
	})
	if err != nil {
		return nil, err
	}

	loadedWb := res.(*Workbench)
	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_READY, 100, "Workbench ready."); err != nil {
		return nil, err
	}

	return loadedWb, nil
}

// loadInspectionData loads the KHI result reader and byte size for the given inspection ID.
func (m *WorkbenchManager) loadInspectionData(inspectionID string) (io.ReadCloser, int64, error) {
	if m.inspectionServer == nil {
		return nil, 0, errors.New("inspection server is not available")
	}
	currentTask := m.inspectionServer.GetInspection(inspectionID)
	if currentTask == nil {
		return nil, 0, fmt.Errorf("inspection %s was not found", inspectionID)
	}
	result, err := currentTask.Result()
	if err != nil {
		return nil, 0, err
	}
	size, err := result.ResultStore.GetInspectionResultSizeInBytes()
	if err != nil {
		return nil, 0, err
	}
	reader, err := result.ResultStore.GetRangeReader(0, int64(size))
	if err != nil {
		return nil, 0, err
	}
	return reader, int64(size), nil
}

// Heartbeat refreshes the lease TTL of an active Workbench session.
func (m *WorkbenchManager) Heartbeat(workbenchID string) (*Workbench, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wb, ok := m.workbenches[workbenchID]
	lease, hasLease := m.leases[workbenchID]
	if !ok || !hasLease || time.Now().After(lease) {
		return nil, time.Time{}, ErrWorkbenchNotFound
	}
	if wb.IsClosed() {
		return nil, time.Time{}, ErrWorkbenchClosed
	}

	expiresAt := time.Now().Add(m.ttl)
	m.leases[workbenchID] = expiresAt
	return wb, expiresAt, nil
}

// GetAndTouch retrieves an active Workbench session and refreshes its lease TTL.
func (m *WorkbenchManager) GetAndTouch(workbenchID string) (*Workbench, error) {
	wb, _, err := m.Heartbeat(workbenchID)
	return wb, err
}

// Close explicitly terminates and frees a Workbench session.
func (m *WorkbenchManager) Close(workbenchID string) error {
	m.mu.Lock()
	wb, ok := m.workbenches[workbenchID]
	delete(m.workbenches, workbenchID)
	delete(m.leases, workbenchID)
	m.mu.Unlock()

	if !ok {
		return ErrWorkbenchNotFound
	}
	wb.Close()
	return nil
}

// Get retrieves an active Workbench without modifying its TTL.
func (m *WorkbenchManager) Get(workbenchID string) (*Workbench, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wb, ok := m.workbenches[workbenchID]
	lease, hasLease := m.leases[workbenchID]
	if !ok || !hasLease || time.Now().After(lease) {
		return nil, ErrWorkbenchNotFound
	}
	if wb.IsClosed() {
		return nil, ErrWorkbenchClosed
	}
	return wb, nil
}

// Leases returns a snapshot copy of current workbench session lease expiration timestamps.
func (m *WorkbenchManager) Leases() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	leases := make(map[string]time.Time, len(m.leases))
	for id, expiresAt := range m.leases {
		leases[id] = expiresAt
	}
	return leases
}

// Remove evicts and closes a workbench session.
func (m *WorkbenchManager) Remove(workbenchID string) {
	m.mu.Lock()
	wb := m.workbenches[workbenchID]
	delete(m.workbenches, workbenchID)
	delete(m.leases, workbenchID)
	m.mu.Unlock()

	if wb != nil {
		wb.Close()
	}
}

// Stop stops the background sweeper and closes all open workbench sessions.
func (m *WorkbenchManager) Stop() {
	if m.sweeper != nil {
		m.sweeper.Stop()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wb := range m.workbenches {
		wb.Close()
	}
	m.workbenches = make(map[string]*Workbench)
	m.leases = make(map[string]time.Time)
}
