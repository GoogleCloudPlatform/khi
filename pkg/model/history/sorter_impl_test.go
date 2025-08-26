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

package history

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"
)

type resourceChunkSortStrategyTestCase struct {
	Name          string
	Chunk         []*Resource
	Parents       []*Resource
	ExpectedChunk []*Resource
	ExpectedError error
}

func testResourceChunkSortStrategy(t *testing.T, name string, sortStrategy ResourceChunkSortStrategy, testCases ...resourceChunkSortStrategyTestCase) {
	t.Run(name, func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				relationship := enum.RelationshipChild
				if len(tc.Chunk) > 0 {
					relationship = tc.Chunk[0].Relationship
				}
				actual, err := sortStrategy.SortChunk(nil, tc.Parents, relationship, tc.Chunk)
				if tc.ExpectedError != nil {
					if !errors.Is(err, tc.ExpectedError) {
						t.Errorf("error is not matching with the expected error.expected:%s,actual:%s", tc.ExpectedError.Error(), err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error %s", err.Error())
					}
					if diff := cmp.Diff(tc.ExpectedChunk, actual); diff != "" {
						t.Errorf("non matching result\n%s", diff)
					}
				}
			})
		}
	})
}

func TestNameSortStrategy(t *testing.T) {
	testResourceChunkSortStrategy(t, "with root layer", NewNameSortStrategy(0, []string{
		"@a", "@b", "@c",
	}), resourceChunkSortStrategyTestCase{
		Name: "simple sort without special keys",
		Chunk: []*Resource{
			newResourceForTesting("c", enum.RelationshipChild),
			newResourceForTesting("b", enum.RelationshipChild),
			newResourceForTesting("a", enum.RelationshipChild),
		},
		Parents: []*Resource{},
		ExpectedChunk: []*Resource{
			newResourceForTesting("a", enum.RelationshipChild),
			newResourceForTesting("b", enum.RelationshipChild),
			newResourceForTesting("c", enum.RelationshipChild),
		},
	},
		resourceChunkSortStrategyTestCase{
			Name: "simple sort with special keys",
			Chunk: []*Resource{
				newResourceForTesting("@c", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
				newResourceForTesting("a", enum.RelationshipChild),
			},
			Parents: []*Resource{},
			ExpectedChunk: []*Resource{
				newResourceForTesting("@c", enum.RelationshipChild),
				newResourceForTesting("a", enum.RelationshipChild),
				newResourceForTesting("b", enum.RelationshipChild),
			},
		},
		resourceChunkSortStrategyTestCase{
			Name: "for different layer",
			Chunk: []*Resource{
				newResourceForTesting("a", enum.RelationshipChild),
			},
			Parents:       []*Resource{newResourceForTesting("parent", enum.RelationshipChild)},
			ExpectedError: ErrorSortSkipped,
		})
}

func TestUnreachableSortStrategy(t *testing.T) {
	testResourceChunkSortStrategy(t, "-", &UnreachableSortStrategy{}, resourceChunkSortStrategyTestCase{
		Name: "simple sort",
		Chunk: []*Resource{
			newResourceForTesting("c", enum.RelationshipChild),
			newResourceForTesting("b", enum.RelationshipChild),
			newResourceForTesting("a", enum.RelationshipChild),
		},
		Parents: []*Resource{},
		ExpectedChunk: []*Resource{
			newResourceForTesting("a", enum.RelationshipChild),
			newResourceForTesting("b", enum.RelationshipChild),
			newResourceForTesting("c", enum.RelationshipChild),
		},
	})
}
