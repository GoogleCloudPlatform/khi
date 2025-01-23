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

import "runtime"

type ResourceMonitor interface {
	GetUsedMemory() int
}

type ResourceMonitorImpl struct {
}

// GetUsedMemory implements ResourceMonitor.
func (r *ResourceMonitorImpl) GetUsedMemory() int {
	// Get server status
	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)
	return int(memStat.Sys)
}

var _ ResourceMonitor = (*ResourceMonitorImpl)(nil)

type ResourceMonitorMock struct {
	UsedMemory int
}

// GetUsedMemory implements ResourceMonitor.
func (r *ResourceMonitorMock) GetUsedMemory() int {
	return r.UsedMemory
}

var _ ResourceMonitor = (*ResourceMonitorMock)(nil)
