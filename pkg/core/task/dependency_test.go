// Copyright 2026 Google LLC
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

package coretask

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

func TestTagReference(t *testing.T) {
	testCases := []struct {
		name            string
		tag             string
		opts            []taskid.FanInOption
		wantCardinality taskid.EdgeCardinality
		wantScope       taskid.DependencyScope
		wantTag         string
	}{
		{
			name:            "default tag reference",
			tag:             "test/tag",
			opts:            nil,
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
			wantTag:         "test/tag",
		},
		{
			name:            "tag reference from active features",
			tag:             "test/features_tag",
			opts:            []taskid.FanInOption{taskid.ScopeActiveFeatures},
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveFeatures,
			wantTag:         "test/features_tag",
		},
		{
			name:            "tag reference from active graph",
			tag:             "test/graph_tag",
			opts:            []taskid.FanInOption{taskid.ScopeActiveGraph},
			wantCardinality: taskid.CardinalityFanIn,
			wantScope:       taskid.ScopeActiveGraph,
			wantTag:         "test/graph_tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := NewTagReference[int](tc.tag, tc.opts...)

			if got := ref.DescriptorCardinality(); got != tc.wantCardinality {
				t.Errorf("DescriptorCardinality() = %v, want %v", got, tc.wantCardinality)
			}
			if got := ref.DescriptorScope(); got != tc.wantScope {
				t.Errorf("DescriptorScope() = %v, want %v", got, tc.wantScope)
			}
			if got := ref.Tag(); got != tc.wantTag {
				t.Errorf("Tag() = %q, want %q", got, tc.wantTag)
			}
			if got := ref.GetZeroValue(); got != 0 {
				t.Errorf("GetZeroValue() = %v, want 0", got)
			}
		})
	}
}

type customDependency struct{}

func (c customDependency) DescriptorCardinality() taskid.EdgeCardinality {
	return taskid.CardinalityPointToPoint
}

func (c customDependency) DescriptorScope() taskid.DependencyScope {
	return taskid.ScopeAll
}

var _ taskid.DependencyDescriptor = (*customDependency)(nil)
