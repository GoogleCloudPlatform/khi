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

package query

import (
	"slices"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var QueryMetadataKey = "query"

type QueryItem struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Query string `json:"query"`
}

type QueryMetadata struct {
	Queries []*QueryItem
	lock    sync.Mutex
}

// Labels implements metadata.Metadata.
func (*QueryMetadata) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInDryRunResult(), metadata.IncludeInRunResult())
}

// ToSerializable implements metadata.Metadata.
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

var _ metadata.Metadata = (*QueryMetadata)(nil)

type QueryMetadataFactory struct{}

// Instanciate implements metadata.MetadataFactory.
func (q *QueryMetadataFactory) Instanciate() metadata.Metadata {
	return &QueryMetadata{}
}

var _ metadata.MetadataFactory = (*QueryMetadataFactory)(nil)
