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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

type QueryItem struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Query          string `json:"query"`
	EstimatedCount *int64 `json:"estimatedCount,omitempty"`
	Incomplete     bool   `json:"incomplete,omitempty"`
	Pending        bool   `json:"pending,omitempty"`
}

type QueryMetadata struct {
	Queries []*QueryItem
	lock    sync.Mutex
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
	q.setQueryInternal(id, name, queryString, nil, false, false)
}

// SetIncompleteQuery sets or updates the query item marked as incomplete without an estimated count.
func (q *QueryMetadata) SetIncompleteQuery(id string, name string, queryString string) {
	q.setQueryInternal(id, name, queryString, nil, true, false)
}

// SetPendingQuery sets or updates the query item marked as pending without an estimated count.
func (q *QueryMetadata) SetPendingQuery(id string, name string, queryString string) {
	q.setQueryInternal(id, name, queryString, nil, false, true)
}

// SetQueryWithEstimate sets or updates the query item along with its estimated log count.
func (q *QueryMetadata) SetQueryWithEstimate(id string, name string, queryString string, estimatedCount int64) {
	q.setQueryInternal(id, name, queryString, &estimatedCount, false, false)
}

func (q *QueryMetadata) setQueryInternal(id string, name string, queryString string, estimatedCount *int64, incomplete bool, pending bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for _, qi := range q.Queries {
		if qi.Id == id {
			qi.Name = name
			qi.Query = queryString
			qi.EstimatedCount = estimatedCount
			qi.Incomplete = incomplete
			qi.Pending = pending
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
	})
}

var _ Metadata = (*QueryMetadata)(nil)

func NewQueryMetadata() *QueryMetadata {
	return &QueryMetadata{
		Queries: []*QueryItem{},
	}
}
