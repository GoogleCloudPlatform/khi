// Copyright 2025 Google LLC
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

package gcp_types

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

type ResourceNamesInput struct {
	resourceNames *typedmap.TypedMap
}

func NewResourceNamesInput() *ResourceNamesInput {
	return &ResourceNamesInput{
		resourceNames: typedmap.NewTypedMap(),
	}
}

type QueryResourceNames struct {
	QueryID              string
	DefaultResourceNames []string
}

func (q *QueryResourceNames) GetInputID() string {
	return fmt.Sprintf("khi.google.com/input/query-resource-names/%s", q.QueryID)
}

func (r *ResourceNamesInput) UpdateDefaultResourceNamesForQuery(queryID string, defaultResourceNames []string) {
	r.ensureQueryID(queryID)
	queryResourceNames := typedmap.GetOrDefault(r.resourceNames, getMapKeyForQueryID(queryID), &QueryResourceNames{})
	queryResourceNames.DefaultResourceNames = defaultResourceNames
}

func (r *ResourceNamesInput) GetResourceNamesForQuery(queryID string) *QueryResourceNames {
	r.ensureQueryID(queryID)
	return typedmap.GetOrDefault(r.resourceNames, getMapKeyForQueryID(queryID), &QueryResourceNames{})
}

func (r *ResourceNamesInput) ensureQueryID(queryID string) {
	_, found := typedmap.Get(r.resourceNames, getMapKeyForQueryID(queryID))
	if !found {
		typedmap.Set(r.resourceNames, getMapKeyForQueryID(queryID), &QueryResourceNames{
			QueryID:              queryID,
			DefaultResourceNames: []string{},
		})
	}
}

// GetQueryResourceNamePairs returns all query ID and resource name pairs.
func (r *ResourceNamesInput) GetQueryResourceNamePairs() []*QueryResourceNames {
	queries := []*QueryResourceNames{}
	for _, queryID := range r.resourceNames.Keys() {
		resourceNames, found := typedmap.Get(r.resourceNames, getMapKeyForQueryID(queryID))
		if !found {
			continue
		}
		queries = append(queries, resourceNames)
	}
	return queries
}

func getMapKeyForQueryID(queryID string) typedmap.TypedKey[*QueryResourceNames] {
	return typedmap.NewTypedKey[*QueryResourceNames](queryID)
}
