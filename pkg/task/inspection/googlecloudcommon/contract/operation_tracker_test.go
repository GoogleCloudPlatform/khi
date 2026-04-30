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

package googlecloudcommon_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestGCPOperationStateTracker(t *testing.T) {
	createReader := func(t *testing.T, data map[string]any) *structured.NodeReader {
		if data == nil {
			return nil
		}
		node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to create node: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	t.Run("Long running operation with response", func(t *testing.T) {
		tracker := NewGCPOperationStateTracker()

		start := &GCPAuditLogFieldSet{
			OperationID:    "op1",
			OperationFirst: true,
			OperationLast:  false,
			Request:        createReader(t, map[string]any{"key": "value"}),
		}

		manifest, shouldUpdate := tracker.TrackAndGetManifest(start)
		if shouldUpdate {
			t.Errorf("Starting log should not trigger update")
		}
		if manifest != "" {
			t.Errorf("Starting log should not return manifest")
		}

		end := &GCPAuditLogFieldSet{
			OperationID:    "op1",
			OperationFirst: false,
			OperationLast:  true,
			Response:       createReader(t, map[string]any{"id": "res1"}),
		}

		manifest, shouldUpdate = tracker.TrackAndGetManifest(end)
		if !shouldUpdate {
			t.Errorf("Ending log should trigger update")
		}
		want := "id: res1\n"
		if diff := cmp.Diff(want, manifest); diff != "" {
			t.Errorf("manifest mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Long running operation fallback to request", func(t *testing.T) {
		tracker := NewGCPOperationStateTracker()

		start := &GCPAuditLogFieldSet{
			OperationID:    "op1",
			OperationFirst: true,
			OperationLast:  false,
			Request:        createReader(t, map[string]any{"key": "value"}),
		}

		tracker.TrackAndGetManifest(start)

		end := &GCPAuditLogFieldSet{
			OperationID:    "op1",
			OperationFirst: false,
			OperationLast:  true,
			// No response
		}

		manifest, shouldUpdate := tracker.TrackAndGetManifest(end)
		if !shouldUpdate {
			t.Errorf("Ending log should trigger update")
		}
		want := "key: value\n"
		if diff := cmp.Diff(want, manifest); diff != "" {
			t.Errorf("manifest mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("Immediate operation", func(t *testing.T) {
		tracker := NewGCPOperationStateTracker()

		imm := &GCPAuditLogFieldSet{
			OperationFirst: true,
			OperationLast:  true,
			Response:       createReader(t, map[string]any{"imm": "res"}),
		}

		manifest, shouldUpdate := tracker.TrackAndGetManifest(imm)
		if !shouldUpdate {
			t.Errorf("Immediate log should trigger update")
		}
		want := "imm: res\n"
		if diff := cmp.Diff(want, manifest); diff != "" {
			t.Errorf("manifest mismatch (-want +got):\n%s", diff)
		}
	})
}
