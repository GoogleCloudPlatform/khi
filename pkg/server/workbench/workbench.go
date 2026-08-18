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

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// Workbench represents an active in-memory analysis workspace for an inspection dataset.
type Workbench struct {
	id           string
	inspectionID string
	mu           sync.RWMutex
	closed       bool

	metadataChunks []*khifilev6.MetadataChunk
	internPool     *khifilev6.InterningPoolChunk
	styleChunk     *khifilev6.TimelineStyleChunk
	logChunks      []*khifilev6.LogChunk
	timelineChunks []*khifilev6.TimelineChunk
}

// NewWorkbench creates a new Workbench instance.
func NewWorkbench(id string, inspectionID string) *Workbench {
	return &Workbench{
		id:           id,
		inspectionID: inspectionID,
	}
}

// ID returns the unique workbench identifier.
func (w *Workbench) ID() string {
	return w.id
}

// InspectionID returns the associated inspection identifier.
func (w *Workbench) InspectionID() string {
	return w.inspectionID
}

// IsClosed checks whether the workbench has been closed.
func (w *Workbench) IsClosed() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.closed
}

// Close marks the workbench as closed and releases in-memory chunk references.
func (w *Workbench) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	w.metadataChunks = nil
	w.internPool = nil
	w.styleChunk = nil
	w.logChunks = nil
	w.timelineChunks = nil
}
