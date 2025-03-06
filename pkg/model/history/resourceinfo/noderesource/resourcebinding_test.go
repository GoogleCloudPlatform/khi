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

package noderesource

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestPodResourceBinding_GetResourcePath(t *testing.T) {
	binding := &PodResourceBinding{
		PodNamespace: "test-namespace",
		PodName:      "test-pod",
	}
	want := resourcepath.Pod("test-namespace", "test-pod")
	got := binding.GetResourcePath()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPodResourceBinding_GetUniqueIdentifier(t *testing.T) {
	binding := &PodResourceBinding{
		PodSandboxID: "test-id",
	}
	want := "test-id"
	got := binding.GetUniqueIdentifier()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPodResourceBinding_RewriteLogSummary(t *testing.T) {
	binding := &PodResourceBinding{
		PodSandboxID: "1234567890",
		PodNamespace: "test-namespace",
		PodName:      "test-pod",
	}
	testCases := []struct {
		name    string
		summary string
		want    string
	}{
		{
			name:    "replace id with readable name",
			summary: "test 1234567890 message",
			want:    "test 1234567...(test-namespace/test-pod) message【test-namespace/test-pod】",
		},
		{
			name:    "without id",
			summary: "test message",
			want:    "test message【test-namespace/test-pod】",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := binding.RewriteLogSummary(tc.summary)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

	}
}

func TestPodResourceBinding_NewContainerResourceBinding(t *testing.T) {
	podBinding := &PodResourceBinding{
		PodSandboxID: "1234567890",
		PodNamespace: "test-namespace",
		PodName:      "test-pod",
	}

	containerBinding := podBinding.NewContainerResourceBinding("abcdefghijk", "test-container")

	want := &ContainerResourceBinding{
		ConainerID:    "abcdefghijk",
		ContainerName: "test-container",
		PodNamespace:  "test-namespace",
		PodName:       "test-pod",
	}

	if diff := cmp.Diff(want, containerBinding); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestContainerResourceBinding_GetResourcePath(t *testing.T) {
	binding := &ContainerResourceBinding{
		PodNamespace:  "test-namespace",
		PodName:       "test-pod",
		ContainerName: "test-container",
	}

	want := resourcepath.Container("test-namespace", "test-pod", "test-container")
	got := binding.GetResourcePath()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestContainerResourceBinding_GetUniqueIdentifier(t *testing.T) {
	binding := &ContainerResourceBinding{
		ConainerID: "test-id",
	}
	want := "test-id"
	got := binding.GetUniqueIdentifier()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestContainerResourceBinding_RewriteLogSummary(t *testing.T) {
	binding := &ContainerResourceBinding{
		ConainerID:    "1234567890",
		PodNamespace:  "test-namespace",
		PodName:       "test-pod",
		ContainerName: "test-container",
	}

	testCases := []struct {
		name    string
		summary string
		want    string
	}{
		{
			name:    "replace id with readable name",
			summary: "test 1234567890 message",
			want:    "test 1234567...(test-container in test-namespace/test-pod) message 【test-container in test-namespace/test-pod】",
		},
		{
			name:    "without id",
			summary: "test message",
			want:    "test message 【test-container in test-namespace/test-pod】",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := binding.RewriteLogSummary(tc.summary)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

	}

}

func TestRewriteIdWithReadableName(t *testing.T) {
	id := "1234567890"
	readableName := "readable-name"

	testCases := []struct {
		name    string
		want    string
		message string
	}{
		{
			name:    "replace id with readable name",
			message: "test message including 1234567890",
			want:    "test message including 1234567...(readable-name)",
		},
		{
			name:    "replace id with readable name with multiple occurances",
			message: "test message including 1234567890 and 1234567890",
			want:    "test message including 1234567...(readable-name) and 1234567...(readable-name)",
		},
		{
			name:    "without id",
			message: "test message",
			want:    "test message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := rewriteIdWithReadableName(id, readableName, tc.message)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
