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
	"testing"

	"github.com/google/go-cmp/cmp"
)

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

func ptr[T any](v T) *T {
	return &v
}

func TestQueryMetadata_SetQueryWithEstimate(t *testing.T) {
	qm := NewQueryMetadata()
	qm.SetQuery("q1", "Query 1", "resource.type=k8s_container")
	qm.SetQueryWithEstimate("q2", "Query 2", "resource.type=k8s_node", 1500)

	// Update existing query q1 with estimate
	qm.SetQueryWithEstimate("q1", "Query 1 Updated", "resource.type=k8s_container updated", 3200)

	expected := []*QueryItem{
		{Id: "q1", Name: "Query 1 Updated", Query: "resource.type=k8s_container updated", EstimatedCount: ptr(int64(3200))},
		{Id: "q2", Name: "Query 2", Query: "resource.type=k8s_node", EstimatedCount: ptr(int64(1500))},
	}

	if diff := cmp.Diff(expected, qm.ToSerializable()); diff != "" {
		t.Errorf("SetQueryWithEstimate mismatch (-want +got):\n%s", diff)
	}
}

func TestQueryMetadata_SetPendingQuery(t *testing.T) {
	qm := NewQueryMetadata()
	qm.SetQuery("q1", "Query 1", "resource.type=k8s_container")
	qm.SetPendingQuery("q2", "Query 2", "resource.type=k8s_node")

	expected := []*QueryItem{
		{Id: "q1", Name: "Query 1", Query: "resource.type=k8s_container"},
		{Id: "q2", Name: "Query 2", Query: "resource.type=k8s_node", Pending: true},
	}

	if diff := cmp.Diff(expected, qm.ToSerializable()); diff != "" {
		t.Errorf("SetPendingQuery mismatch (-want +got):\n%s", diff)
	}
}
