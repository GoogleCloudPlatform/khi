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

	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceMonitor provides methods to monitor server resources.
type ResourceMonitor interface {
	// GetUsedMemory returns the current memory usage of the server process in bytes.
	GetUsedMemory() uint64

	// GetTotalMemory returns the total physical memory of the server in bytes.
	GetTotalMemory() uint64
}

// ResourceMonitorImpl is the real implementation of ResourceMonitor.
type ResourceMonitorImpl struct {
	totalMemory uint64
	once        sync.Once
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
			slog.Error("Failed to get total memory", "error", err)
		}
	})
	return r.totalMemory
}

var _ ResourceMonitor = &ResourceMonitorImpl{}

// ResourceMonitorMock is a mock implementation of ResourceMonitor for testing.
type ResourceMonitorMock struct {
	UsedMemory  uint64
	TotalMemory uint64
}

// GetUsedMemory returns the mocked used memory.
func (r *ResourceMonitorMock) GetUsedMemory() uint64 {
	return r.UsedMemory
}

// GetTotalMemory returns the mocked total memory.
func (r *ResourceMonitorMock) GetTotalMemory() uint64 {
	return r.TotalMemory
}

var _ ResourceMonitor = &ResourceMonitorMock{}
