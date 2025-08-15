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

var QueryMetadataKey = NewMetadataKey[*QueryMetadata]("query")

type QueryItem struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Query string `json:"query"`
}

type QueryMetadata struct {
	Queries []*QueryItem
	lock    sync.Mutex
}

// Labels implements Metadata.
func (*QueryMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInDryRunResult(), IncludeInRunResult())
}

// ToSerializable implements Metadata.
func (q *QueryMetadata) ToSerializable() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	slices.SortFunc(q.Queries, func(a, b *QueryItem) int { return strings.Compare(a.Id, b.Id) })
	return q.Queries
}

func (q *QueryMetadata) SetQuery(id string, name string, queryString string) {
	q.lock.Lock()
	defer q.lock.Unlock()
	for _, qi := range q.Queries {
		if qi.Id == id {
			qi.Name = name
			qi.Query = queryString
			return
		}
	}
	q.Queries = append(q.Queries, &QueryItem{
		Id:    id,
		Name:  name,
		Query: queryString,
	})
}

var _ Metadata = (*QueryMetadata)(nil)

func NewQueryMetadata() *QueryMetadata {
	return &QueryMetadata{
		Queries: []*QueryItem{},
	}
}
