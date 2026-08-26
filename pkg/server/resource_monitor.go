// Copyright 2024 Google LLC
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

package server

import (
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceMonitor provides methods to monitor server resources.
type ResourceMonitor interface {
	// GetUsedMemory returns the current memory usage of the server process in bytes.
	GetUsedMemory() uint64

	// GetTotalMemory returns the total physical memory of the server in bytes.
	GetTotalMemory() uint64

	// GetCPUUsage returns the current CPU usage percentage across all cores (0.0 to 100.0).
	GetCPUUsage() float64
}

// ResourceMonitorImpl is the real implementation of ResourceMonitor.
type ResourceMonitorImpl struct {
	totalMemory uint64
	once        sync.Once

	cpuMu       sync.RWMutex
	cpuUsage    float64
	initialized bool
	closeChan   chan struct{}
	closeOnce   sync.Once
}

var _ ResourceMonitor = (*ResourceMonitorImpl)(nil)

// NewResourceMonitorImpl creates and initializes a ResourceMonitorImpl, starting a background CPU sampling loop.
func NewResourceMonitorImpl() *ResourceMonitorImpl {
	r := &ResourceMonitorImpl{
		initialized: true,
		closeChan:   make(chan struct{}),
	}
	go r.startCPUUsageLoop()
	return r
}

func (r *ResourceMonitorImpl) startCPUUsageLoop() {
	// Prime the CPU counter so the next ticker tick calculates differences.
	_, _ = cpu.Percent(0, false)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.closeChan:
			return
		case <-ticker.C:
			percents, err := cpu.Percent(0, false)
			if err == nil && len(percents) > 0 {
				r.cpuMu.Lock()
				r.cpuUsage = percents[0]
				r.cpuMu.Unlock()
			} else if err != nil {
				slog.Error("failed to get CPU usage", "error", err)
			}
		}
	}
}

// Close terminates the background CPU monitoring loop.
func (r *ResourceMonitorImpl) Close() {
	if !r.initialized {
		return
	}
	r.closeOnce.Do(func() {
		close(r.closeChan)
	})
}

// GetUsedMemory returns the current memory usage using runtime.MemStats (Alloc).
func (r *ResourceMonitorImpl) GetUsedMemory() uint64 {
	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)
	return memStat.Alloc
}

// GetTotalMemory returns the total physical memory using gopsutil.
// The result is cached after the first call.
func (r *ResourceMonitorImpl) GetTotalMemory() uint64 {
	r.once.Do(func() {
		v, err := mem.VirtualMemory()
		if err == nil {
			r.totalMemory = v.Total
		} else {
			slog.Error("failed to get total memory", "error", err)
		}
	})
	return r.totalMemory
}

// GetCPUUsage returns the latest sampled CPU usage percentage from the background loop.
func (r *ResourceMonitorImpl) GetCPUUsage() float64 {
	if !r.initialized {
		panic("ResourceMonitorImpl must be created using NewResourceMonitorImpl")
	}
	r.cpuMu.RLock()
	defer r.cpuMu.RUnlock()
	return r.cpuUsage
}

// ResourceMonitorMock is a mock implementation of ResourceMonitor for testing.
type ResourceMonitorMock struct {
	UsedMemory  uint64
	TotalMemory uint64
	CPUUsage    float64
}

var _ ResourceMonitor = (*ResourceMonitorMock)(nil)

// GetUsedMemory returns the mocked used memory.
func (r *ResourceMonitorMock) GetUsedMemory() uint64 {
	return r.UsedMemory
}

// GetTotalMemory returns the mocked total memory.
func (r *ResourceMonitorMock) GetTotalMemory() uint64 {
	return r.TotalMemory
}

// GetCPUUsage returns the mocked CPU usage.
func (r *ResourceMonitorMock) GetCPUUsage() float64 {
	return r.CPUUsage
}
