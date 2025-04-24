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

// LoggingFilterResourceNameStore stores resource names for each Cloud Logging query tasks.
type LoggingFilterResourceNameStore struct {
	resourceNames *typedmap.TypedMap
}

func NewLoggingFilterResourceNameStore() *LoggingFilterResourceNameStore {
	return &LoggingFilterResourceNameStore{
		resourceNames: typedmap.NewTypedMap(),
	}
}

type LoggingFilterResourceName struct {
	FilterID             string
	FilterName           string
	DefaultResourceNames []string
}

func (q *LoggingFilterResourceName) GetInputID() string {
	return fmt.Sprintf("cloud.google.com/input/query-resource-names/%s", q.FilterID)
}

func (r *LoggingFilterResourceNameStore) UpdateDefaultResourceNamesForLoggingFilter(loggingFilterID string, loggingFilterName string, defaultResourceNames []string) {
	_, found := typedmap.Get(r.resourceNames, getMapKeyForLoggingFilterID(loggingFilterID))
	if !found {
		typedmap.Set(r.resourceNames, getMapKeyForLoggingFilterID(loggingFilterID), &LoggingFilterResourceName{
			FilterID:             loggingFilterID,
			FilterName:           loggingFilterName,
			DefaultResourceNames: []string{},
		})
	}
	queryResourceNames := typedmap.GetOrDefault(r.resourceNames, getMapKeyForLoggingFilterID(loggingFilterID), &LoggingFilterResourceName{})
	queryResourceNames.DefaultResourceNames = defaultResourceNames
}

func (r *LoggingFilterResourceNameStore) GetLoggingFilterResourceName(loggingFilterID string) *LoggingFilterResourceName {
	return typedmap.GetOrDefault(r.resourceNames, getMapKeyForLoggingFilterID(loggingFilterID), &LoggingFilterResourceName{})
}

// GetLoggingFilterResourceNames returns all query ID and resource name pairs.
func (r *LoggingFilterResourceNameStore) GetLoggingFilterResourceNames() []*LoggingFilterResourceName {
	result := []*LoggingFilterResourceName{}
	for _, filterID := range r.resourceNames.Keys() {
		resourceNames, found := typedmap.Get(r.resourceNames, getMapKeyForLoggingFilterID(filterID))
		if !found {
			continue
		}
		result = append(result, resourceNames)
	}
	return result
}

func getMapKeyForLoggingFilterID(loggingFilterID string) typedmap.TypedKey[*LoggingFilterResourceName] {
	return typedmap.NewTypedKey[*LoggingFilterResourceName](loggingFilterID)
}
