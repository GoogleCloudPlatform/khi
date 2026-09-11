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

package inspectioncore_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/google/go-cmp/cmp"
)

// TestFeatureTaskLabels tests FeatureTaskLabel.
func TestFeatureTaskLabels(t *testing.T) {
	t.Run("FeatureTaskLabel", func(t *testing.T) {
		labelOpt := FeatureTaskLabel(
			"title",
			"description",
			100,
			true,
		)
		label := coretask.NewLabelSet(labelOpt)

		type expectations struct {
			FeatureFlag        bool
			Title              string
			Description        string
			Order              int
			DefaultFeatureFlag bool
		}

		got := expectations{
			FeatureFlag:        typedmap.GetOrDefault(label, LabelKeyInspectionFeatureFlag, false),
			Title:              typedmap.GetOrDefault(label, LabelKeyFeatureTaskTitle, ""),
			Description:        typedmap.GetOrDefault(label, LabelKeyFeatureTaskDescription, ""),
			Order:              typedmap.GetOrDefault(label, LabelKeyFeatureTaskOrder, 0),
			DefaultFeatureFlag: typedmap.GetOrDefault(label, LabelKeyInspectionDefaultFeatureFlag, false),
		}

		want := expectations{
			FeatureFlag:        true,
			Title:              "title",
			Description:        "description",
			Order:              100,
			DefaultFeatureFlag: true,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("FeatureTaskLabel label mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestLabelSelector_Match tests LabelSelector.Match.
func TestLabelSelector_Match(t *testing.T) {
	tests := []struct {
		name         string
		selector     LabelSelector
		targetLabels map[string]string
		want         bool
	}{
		{
			name: "Exact match",
			selector: LabelSelector{
				"env": "gcp",
			},
			targetLabels: map[string]string{
				"env": "gcp",
			},
			want: true,
		},
		{
			name: "Target has superset of labels",
			selector: LabelSelector{
				"env": "gcp",
			},
			targetLabels: map[string]string{
				"env":      "gcp",
				"platform": "k8s",
			},
			want: true,
		},
		{
			name: "Value mismatch",
			selector: LabelSelector{
				"env": "gcp",
			},
			targetLabels: map[string]string{
				"env": "aws",
			},
			want: false,
		},
		{
			name: "Target missing key",
			selector: LabelSelector{
				"env": "gcp",
			},
			targetLabels: map[string]string{
				"platform": "k8s",
			},
			want: false,
		},
		{
			name:         "Empty selector matches any target labels",
			selector:     LabelSelector{},
			targetLabels: map[string]string{"env": "gcp"},
			want:         true,
		},
		{
			name: "Selector with keys fails on nil target labels",
			selector: LabelSelector{
				"env": "gcp",
			},
			targetLabels: nil,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.selector.Match(tt.targetLabels)
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInspectionTypeLabelSelector tests InspectionTypeLabelSelector.
func TestInspectionTypeLabelSelector(t *testing.T) {
	labelOpt := InspectionTypeLabelSelector(map[string]string{
		"env": "gcp",
	})
	labelSet := coretask.NewLabelSet(labelOpt)

	got, ok := typedmap.Get(labelSet, LabelKeyInspectionTypeLabelSelector)
	if !ok {
		t.Fatalf("LabelKeyInspectionTypeLabelSelector not found in label set")
	}

	want := LabelSelector{
		"env": "gcp",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("InspectionTypeLabelSelector mismatch (-want +got):\n%s", diff)
	}
}
