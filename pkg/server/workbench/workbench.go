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
	"errors"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
)

var (
	// ErrStructNotFound indicates that the requested struct ID was not found in the intern pool.
	ErrStructNotFound = errors.New("struct not found")
)

// Workbench represents an active in-memory analysis workspace for an inspection dataset.
type Workbench struct {
	id           string
	inspectionID string
	mu           sync.RWMutex
	closed       bool

	metadataChunks []*khifilev6.MetadataChunk
	internPool     *khifilev6model.InternPool
	styleChunk     *khifilev6.TimelineStyleChunk
	logChunks      []*khifilev6.LogChunk
	timelineChunks []*khifilev6.TimelineChunk
	searchIndex    *SearchIndex
}

// NewWorkbench creates a new Workbench instance.
func NewWorkbench(id string, inspectionID string) *Workbench {
	return &Workbench{
		id:           id,
		inspectionID: inspectionID,
		internPool:   khifilev6model.NewInternPool(nil),
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

// ReadStructYAML decodes the interned struct matching the given structID and returns its YAML string representation.
func (w *Workbench) ReadStructYAML(structID uint32) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return "", ErrWorkbenchClosed
	}

	if w.internPool == nil {
		return "", ErrStructNotFound
	}

	s := w.internPool.ResolveStructFromID(structID)
	if s == nil {
		return "", ErrStructNotFound
	}

	node, err := khifilev6model.FromInternedStruct(s, w.internPool)
	if err != nil {
		return "", fmt.Errorf("failed to decode interned struct: %w", err)
	}

	yamlBytes, err := (&structured.YAMLNodeSerializer{}).Serialize(node)
	if err != nil {
		return "", fmt.Errorf("failed to serialize node to YAML: %w", err)
	}

	return string(yamlBytes), nil
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
	w.searchIndex = nil
}
