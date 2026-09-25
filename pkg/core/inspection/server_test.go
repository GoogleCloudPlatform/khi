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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TestRegisterImportedInspection_NameReservation verifies auto-numbering and reservation when importing inspections.
func TestRegisterImportedInspection_NameReservation(t *testing.T) {
	testCases := []struct {
		name                string
		importInspections   []string
		wantInspectionNames []string
		wantSuggestedFiles  []string
	}{
		{
			name:                "sequential imported inspections with the same name get auto-numbered and reserved",
			importInspections:   []string{"Google Kubernetes Engine", "Google Kubernetes Engine", "Google Kubernetes Engine"},
			wantInspectionNames: []string{"Google Kubernetes Engine", "Google Kubernetes Engine(1)", "Google Kubernetes Engine(2)"},
			wantSuggestedFiles:  []string{"Google Kubernetes Engine.khi", "Google Kubernetes Engine(1).khi", "Google Kubernetes Engine(2).khi"},
		},
		{
			name:                "empty inspection name falls back to Inspection and auto-numbers",
			importInspections:   []string{"", "   "},
			wantInspectionNames: []string{"Inspection", "Inspection(1)"},
			wantSuggestedFiles:  []string{"Inspection.khi", "Inspection(1).khi"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore.IOConfig{})
			if err != nil {
				t.Fatalf("failed to create inspection task server: %v", err)
			}

			for i, baseName := range tc.importInspections {
				id := fmt.Sprintf("imported-%d", i)
				md := typedmap.NewTypedMap()
				header := &inspectionmetadata.HeaderMetadata{
					InspectionName: baseName,
				}
				typedmap.Set(md, inspectionmetadata.HeaderMetadataKey, header)

				runner := server.RegisterImportedInspection(id, nil, md.AsReadonly())
				if runner == nil {
					t.Fatalf("RegisterImportedInspection returned nil")
				}

				wantName := tc.wantInspectionNames[i]
				if header.InspectionName != wantName {
					t.Errorf("imported inspection [%d] InspectionName = %q, want %q", i, header.InspectionName, wantName)
				}
				wantFile := tc.wantSuggestedFiles[i]
				if header.SuggestedFileName != wantFile {
					t.Errorf("imported inspection [%d] SuggestedFileName = %q, want %q", i, header.SuggestedFileName, wantFile)
				}

				if err := server.InspectionNameRegistry().ReserveName("other-id", wantName); err != inspectioncore.ErrInspectionNameAlreadyInUse {
					t.Errorf("expected %q to be reserved, but got error: %v", wantName, err)
				}
			}
		})
	}
}
