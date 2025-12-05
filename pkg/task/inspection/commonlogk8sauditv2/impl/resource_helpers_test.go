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

package commonlogk8sauditv2_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestGetDeletionGracePeriodSeconds(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      int
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
metadata:
  deletionGracePeriodSeconds: 30
`,
			want:      30,
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
metadata:
  name: foo
`,
			want:      0,
			wantFound: false,
		},
		{
			name: "zero",
			yaml: `
metadata:
  deletionGracePeriodSeconds: 0
`,
			want:      0,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetDeletionGracePeriodSeconds(reader)
			if got != tt.want {
				t.Errorf("GetDeletionGracePeriodSeconds() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetDeletionGracePeriodSeconds() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetDeletionTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      string
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
metadata:
  deletionTimestamp: "2024-01-01T00:00:00Z"
`,
			want:      "2024-01-01T00:00:00Z",
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
metadata:
  name: foo
`,
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetDeletionTimestamp(reader)
			if got != tt.want {
				t.Errorf("GetDeletionTimestamp() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetDeletionTimestamp() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetFinalizers(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      []string
		wantFound bool
	}{
		{
			name: "metadata finalizers",
			yaml: `
metadata:
  finalizers:
  - foo
  - bar
`,
			want:      []string{"foo", "bar"},
			wantFound: true,
		},
		{
			name: "spec finalizers",
			yaml: `
spec:
  finalizers:
  - baz
`,
			want:      []string{"baz"},
			wantFound: true,
		},
		{
			name: "both (metadata priority)",
			yaml: `
metadata:
  finalizers:
  - foo
spec:
  finalizers:
  - bar
`,
			want:      []string{"foo"},
			wantFound: true,
		},
		{
			name: "empty list",
			yaml: `
metadata:
  finalizers: []
`,
			want:      nil,
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
metadata:
  name: foo
`,
			want:      nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetFinalizers(reader)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetFinalizers() mismatch (-want +got):\n%s", diff)
			}
			if found != tt.wantFound {
				t.Errorf("GetFinalizers() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetPodPhase(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      string
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
status:
  phase: Running
`,
			want:      "Running",
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
status:
  message: foo
`,
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetPodPhase(reader)
			if got != tt.want {
				t.Errorf("GetPodPhase() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetPodPhase() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetUID(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      string
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
metadata:
  uid: "test-uid"
`,
			want:      "test-uid",
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
metadata:
  name: foo
`,
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetUID(reader)
			if got != tt.want {
				t.Errorf("GetUID() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetUID() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetNodeNameOfPod(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      string
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
spec:
  nodeName: "test-node"
`,
			want:      "test-node",
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
spec:
  containers: []
`,
			want:      "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetNodeNameOfPod(reader)
			if got != tt.want {
				t.Errorf("GetNodeNameOfPod() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetNodeNameOfPod() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestGetCreationTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		want      time.Time
		wantFound bool
	}{
		{
			name: "exists",
			yaml: `
metadata:
  creationTimestamp: "2024-01-01T00:00:00Z"
`,
			want:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantFound: true,
		},
		{
			name: "not exists",
			yaml: `
metadata:
  name: foo
`,
			want:      time.Time{},
			wantFound: false,
		},
		{
			name: "invalid format",
			yaml: `
metadata:
  creationTimestamp: "invalid"
`,
			want:      time.Time{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := mustParseYAML(t, tt.yaml)
			got, found := GetCreationTimestamp(reader)
			if !got.Equal(tt.want) {
				t.Errorf("GetCreationTimestamp() got = %v, want %v", got, tt.want)
			}
			if found != tt.wantFound {
				t.Errorf("GetCreationTimestamp() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func mustParseYAML(t *testing.T, yamlStr string) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromYAML(yamlStr)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	return structured.NewNodeReader(node)
}
