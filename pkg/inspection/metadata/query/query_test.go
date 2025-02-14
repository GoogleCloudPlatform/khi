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
	"testing"

	metadata_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/metadata"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestQueryConformance(t *testing.T) {
	metadata_test.ConformanceMetadataTypeTest(t, &QueryMetadata{
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

func TestQuerySerializeInSortedOrder(t *testing.T) {
	query := QueryMetadata{
		Queries: []*QueryItem{
			{Id: "a"},
			{Id: "c"},
			{Id: "b"},
			{Id: "e"},
			{Id: "d"},
		},
	}

	expected := []*QueryItem{
		{Id: "a"},
		{Id: "b"},
		{Id: "c"},
		{Id: "d"},
		{Id: "e"},
	}
	if diff := cmp.Diff(query.ToSerializable(), expected); diff != "" {
		t.Errorf("Query info serialization result was not in the sorted order\n%s", diff)
	}
}
