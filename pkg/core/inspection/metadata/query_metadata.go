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

package inspectionmetadata

import (
	"slices"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

var _ Metadata = (*QueryMetadata)(nil)

// QueryItem represents a single log query recorded during inspection.
type QueryItem struct {
	// Id is the unique identifier of the query.
	Id string `json:"id"`
	// Name is the human-readable display name of the query.
	Name string `json:"name"`
	// Query is the query string sent to the logging backend.
	Query string `json:"query"`
	// EstimatedCount is the estimated number of matching log entries, if available.
	EstimatedCount *int64 `json:"estimatedCount,omitempty"`
	// Incomplete indicates whether the query string could not be fully constructed.
	Incomplete bool `json:"incomplete,omitempty"`
	// Pending indicates whether the query estimation is still in progress.
	Pending bool `json:"pending,omitempty"`
	// Preset is the fallback estimation tier when an exact count is unavailable.
	Preset logestimator.EstimatedCountPreset `json:"preset,omitempty"`
}

// QueryMetadata stores the set of log queries executed or planned by an inspection.
type QueryMetadata struct {
	// Queries holds the list of recorded query items.
	Queries []*QueryItem
	lock    sync.Mutex
}

// NewQueryMetadata creates a new empty QueryMetadata instance.
func NewQueryMetadata() *QueryMetadata {
	return &QueryMetadata{
		Queries: []*QueryItem{},
	}
}

// Labels implements Metadata.
func (*QueryMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInDryRunResult(), IncludeInRunResult(), IncludeInResultBinary())
}

// ToSerializable implements Metadata.
func (q *QueryMetadata) ToSerializable() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	slices.SortFunc(q.Queries, func(a, b *QueryItem) int { return strings.Compare(a.Id, b.Id) })
	return q.Queries
}

// SetQuery sets or updates the query item with no estimated count (nil).
func (q *QueryMetadata) SetQuery(id string, name string, queryString string) {
	q.setQueryInternal(id, name, queryString, nil, false, false, logestimator.EstimatedCountPresetNone)
}

// SetIncompleteQuery sets or updates the query item marked as incomplete without an estimated count.
func (q *QueryMetadata) SetIncompleteQuery(id string, name string, queryString string) {
	q.setQueryInternal(id, name, queryString, nil, true, false, logestimator.EstimatedCountPresetNone)
}

// SetPendingQuery sets or updates the query item marked as pending without an estimated count.
func (q *QueryMetadata) SetPendingQuery(id string, name string, queryString string) {
	q.setQueryInternal(id, name, queryString, nil, false, true, logestimator.EstimatedCountPresetNone)
}

// SetQueryWithEstimate sets or updates the query item along with its estimated log count.
func (q *QueryMetadata) SetQueryWithEstimate(id string, name string, queryString string, estimatedCount int64) {
	q.setQueryInternal(id, name, queryString, &estimatedCount, false, false, logestimator.EstimatedCountPresetNone)
}

// SetQueryWithPreset sets or updates the query item along with a preset estimation tier.
func (q *QueryMetadata) SetQueryWithPreset(id string, name string, queryString string, preset logestimator.EstimatedCountPreset) {
	q.setQueryInternal(id, name, queryString, nil, false, false, preset)
}

func (q *QueryMetadata) setQueryInternal(id string, name string, queryString string, estimatedCount *int64, incomplete bool, pending bool, preset logestimator.EstimatedCountPreset) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for _, qi := range q.Queries {
		if qi.Id == id {
			qi.Name = name
			qi.Query = queryString
			qi.EstimatedCount = estimatedCount
			qi.Incomplete = incomplete
			qi.Pending = pending
			qi.Preset = preset
			return
		}
	}
	q.Queries = append(q.Queries, &QueryItem{
		Id:             id,
		Name:           name,
		Query:          queryString,
		EstimatedCount: estimatedCount,
		Incomplete:     incomplete,
		Pending:        pending,
		Preset:         preset,
	})
}
