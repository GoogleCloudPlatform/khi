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

package inspectionmetadata

import (
	"slices"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// TimelineExampleNameLimit is the maximum number of example names recorded for each timeline type.
const TimelineExampleNameLimit = 5

// FilterVocabulary lists the values that filter CEL can match in one inspection.
type FilterVocabulary struct {
	TimelineTypes []TimelineTypeVocabulary `json:"timelineTypes"`
	LogTypes      []string                 `json:"logTypes"`
}

// TimelineTypeVocabulary summarizes the timeline names of one timeline type.
type TimelineTypeVocabulary struct {
	Label        string   `json:"label"`
	NameCount    int      `json:"nameCount"`
	ExampleNames []string `json:"exampleNames"`
}

// PathKey returns the key of path in timeline CEL, which is the lowercased label.
func (v TimelineTypeVocabulary) PathKey() string {
	return strings.ToLower(v.Label)
}

func cloneFilterVocabulary(v FilterVocabulary) FilterVocabulary {
	timelineTypes := make([]TimelineTypeVocabulary, len(v.TimelineTypes))
	for i, tt := range v.TimelineTypes {
		timelineTypes[i] = TimelineTypeVocabulary{
			Label:        tt.Label,
			NameCount:    tt.NameCount,
			ExampleNames: slices.Clone(tt.ExampleNames),
		}
	}
	return FilterVocabulary{
		TimelineTypes: timelineTypes,
		LogTypes:      slices.Clone(v.LogTypes),
	}
}

// FilterVocabularyMetadata holds the FilterVocabulary recorded when the inspection result is serialized.
type FilterVocabularyMetadata struct {
	mu         sync.RWMutex
	vocabulary FilterVocabulary
}

var _ Metadata = (*FilterVocabularyMetadata)(nil)

// NewFilterVocabularyMetadata creates an empty FilterVocabularyMetadata.
func NewFilterVocabularyMetadata() *FilterVocabularyMetadata {
	return &FilterVocabularyMetadata{}
}

// Set stores a copy of the given filter vocabulary.
func (m *FilterVocabularyMetadata) Set(v FilterVocabulary) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vocabulary = cloneFilterVocabulary(v)
}

// Get returns a copy of the stored filter vocabulary.
func (m *FilterVocabularyMetadata) Get() FilterVocabulary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneFilterVocabulary(m.vocabulary)
}

// Labels returns the metadata labels for FilterVocabularyMetadata.
func (*FilterVocabularyMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInResultBinary())
}

// ToSerializable returns a snapshot of the filter vocabulary for serialization.
func (m *FilterVocabularyMetadata) ToSerializable() any {
	return m.Get()
}
