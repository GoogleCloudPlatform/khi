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

package enum

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParentRelationshipMetadataIsFilled(t *testing.T) {
	for i := 0; i <= int(relationshipUnusedEnd); i++ {
		if _, ok := ParentRelationships[ParentRelationship(i)]; !ok {
			t.Errorf("ParentRelationshipMetadata[%d] is not filled", i)
		}
	}
}

func TestParentRelationshipMetadataIsValid(t *testing.T) {
	for i := 0; i <= int(relationshipUnusedEnd); i++ {
		if relationship, ok := ParentRelationships[ParentRelationship(i)]; ok {
			t.Run(fmt.Sprintf("%d-%s", i, relationship.EnumKeyName), func(t *testing.T) {
				if relationship.EnumKeyName == "" {
					t.Errorf("EnumKeyName in `%s(%d)` is empty", relationship.Label, i)
				}
				if relationship.Visible {
					if relationship.LabelColor == "" {
						t.Errorf("LabelColor in `%s(%d)` is empty", relationship.Label, i)
					}
					if relationship.LabelBackgroundColor == "" {
						t.Errorf("LabelBackgroundColor in `%s(%d)` is empty", relationship.Label, i)
					}
					if relationship.Hint == "" {
						t.Errorf("Hint in `%s(%d)` is empty", relationship.Label, i)
					}
					if strings.Contains(relationship.Label, " ") {
						t.Errorf("Label in `%s(%d)` contains space(label must be valid as css class)", relationship.Label, i)
					}
				}
			})
		}
	}
}

func TestParentRelationshipOrderIsValid(t *testing.T) {
	testCase := []struct {
		name          string
		expectedOrder []ParentRelationship
	}{
		{
			name: "order of node subresources",
			expectedOrder: []ParentRelationship{
				RelationshipChild,
				RelationshipSerialPort,
				RelationshipResourceCondition,
				RelationshipOperation,
				RelationshipNodeComponent,
				RelationshipOwnerReference,
				RelationshipPodBinding,
			},
		},
		{
			name: "order of pod subresources",
			expectedOrder: []ParentRelationship{
				RelationshipChild,
				RelationshipResourceCondition,
				RelationshipContainer,
				RelationshipEndpointSlice,
				RelationshipNetworkEndpointGroup,
			},
		},
	}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			original := slices.Clone(tc.expectedOrder)
			slices.Reverse(original)
			slices.SortFunc(original, func(a, b ParentRelationship) int {
				return ParentRelationships[a].SortPriority - ParentRelationships[b].SortPriority
			})
			if diff := cmp.Diff(tc.expectedOrder, original); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}
