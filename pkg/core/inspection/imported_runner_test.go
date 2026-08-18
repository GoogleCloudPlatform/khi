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

package coreinspection

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestNewImportedInspectionRunner(t *testing.T) {
	testCases := []struct {
		name             string
		id               string
		inspectionName   string
		wantStarted      bool
		wantMetadataName string
	}{
		{
			name:             "successfully creates imported runner with completed state",
			id:               "imported-123",
			inspectionName:   "imported-cluster",
			wantStarted:      true,
			wantMetadataName: "imported-cluster",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ioConfig := &inspectioncore_contract.IOConfig{
				TemporaryFolder: t.TempDir(),
			}
			server, err := NewServer(ioConfig)
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}

			dummyStore := inspectioncore_contract.NewFileSystemInspectionResultRepository(t.TempDir() + "/dummy.khi")
			metadataMap := typedmap.NewTypedMap()
			header := &inspectionmetadata.HeaderMetadata{
				InspectionName: tc.inspectionName,
				InspectionType: "imported",
			}
			typedmap.Set(metadataMap, inspectionmetadata.HeaderMetadataKey, header)

			runner := server.RegisterImportedInspection(tc.id, dummyStore, metadataMap.AsReadonly())
			if runner.Started() != tc.wantStarted {
				t.Errorf("Started() mismatch: got %v, want %v", runner.Started(), tc.wantStarted)
			}

			gotRunner := server.GetInspection(tc.id)
			if gotRunner == nil {
				t.Fatalf("GetInspection(%s) returned nil", tc.id)
			}

			result, err := gotRunner.Result()
			if err != nil {
				t.Fatalf("Result() failed: %v", err)
			}
			if result.ResultStore != dummyStore {
				t.Errorf("ResultStore mismatch: got %v, want %v", result.ResultStore, dummyStore)
			}

			curMetadata, err := gotRunner.GetCurrentMetadata()
			if err != nil {
				t.Fatalf("GetCurrentMetadata() failed: %v", err)
			}
			gotHeader, found := typedmap.Get(curMetadata, inspectionmetadata.HeaderMetadataKey)
			if !found {
				t.Fatal("HeaderMetadata not found in CurrentMetadata")
			}
			if diff := cmp.Diff(tc.wantMetadataName, gotHeader.InspectionName); diff != "" {
				t.Errorf("InspectionName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
