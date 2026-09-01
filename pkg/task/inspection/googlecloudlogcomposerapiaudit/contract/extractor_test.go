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

package googlecloudlogcomposerapiaudit_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestExtractComposerAuditLogResource(t *testing.T) {
	testCases := []struct {
		name    string
		input   map[string]any
		want    ComposerAuditLogResourceFieldSet
		wantErr bool
	}{
		{
			name: "full resource labels",
			input: map[string]any{
				"resource": map[string]any{
					"type": "cloud_composer_environment",
					"labels": map[string]any{
						"project_id":       "test-project",
						"location":         "us-central1",
						"environment_name": "cluster-8rlk9",
					},
				},
			},
			want: ComposerAuditLogResourceFieldSet{
				EnvironmentName: "cluster-8rlk9",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			wantErr: false,
		},
		{
			name: "missing labels default to unknown",
			input: map[string]any{
				"resource": map[string]any{
					"type":   "cloud_composer_environment",
					"labels": map[string]any{},
				},
			},
			want: ComposerAuditLogResourceFieldSet{
				EnvironmentName: "unknown",
				Location:        "unknown",
				ProjectID:       "unknown",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromGoValue(tc.input, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				t.Fatalf("failed to create structured node: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			got, err := ExtractComposerAuditLogResource(nodeReader)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ExtractComposerAuditLogResource() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractComposerAuditLogResource() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := ComposerAuditLogResourceFieldSet{
			EnvironmentName: "mock-env",
			Location:        "mock-loc",
			ProjectID:       "mock-proj",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractComposerAuditLogResource(reader)
		if err != nil {
			t.Fatalf("ExtractComposerAuditLogResource() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractComposerAuditLogResource() mismatch (-want +got):\n%s", diff)
		}
	})
}
