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
	"encoding/json"
	"testing"
)

func ConformanceMetadataTypeTest(t *testing.T, m Metadata) {
	t.Run("metadata type must be serializable", func(t *testing.T) {
		ConformanceTestMetadataIsSerializable(t, m)
	})
}

func ConformanceTestMetadataIsSerializable(t *testing.T, m Metadata) {
	_, err := json.Marshal(m.ToSerializable())
	if err != nil {
		t.Errorf("Expected metadata is JSON serializable. But returned an error\n%v", err)
	}
}

func TestHeaderMetadataConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, &HeaderMetadata{})
}

func TestConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, NewLogMetadata())
}

func TestErrorMetadataConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, &ErrorMessageSetMetadata{
		[]*ErrorMessage{
			{},
		},
	})
}

func TestQueryMetadataConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, &QueryMetadata{
		Queries: []*QueryItem{
			{
				Id:    "foo",
				Query: "foo-body",
			},
			{
				Id:    "bar",
				Query: "bar-body",
			},
		},
	})
}

func newProgressforConformanceTest() Metadata {
	progress := NewProgress()
	progress.GetOrCreateTaskProgress("foo")
	progress.GetOrCreateTaskProgress("bar")
	return progress
}

func TestProgressConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, newProgressforConformanceTest())
}

func TestPlanMetadataConformance(t *testing.T) {
	ConformanceMetadataTypeTest(t, &InspectionPlanMetadata{})
}
