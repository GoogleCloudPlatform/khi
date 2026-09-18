package coreinspection

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestGetDefaultInspectionName(t *testing.T) {
	testCases := []struct {
		name                string
		inspectionTypeID    string
		existingInspections []struct {
			id   string
			name string
		}
		excludeID string
		want      string
	}{
		{
			name:             "no existing inspections returns base inspection type name",
			inspectionTypeID: "gke",
			want:             "Google Kubernetes Engine",
		},
		{
			name:             "base name already exists returns suffix (1)",
			inspectionTypeID: "gke",
			existingInspections: []struct {
				id   string
				name string
			}{
				{id: "insp-1", name: "Google Kubernetes Engine"},
			},
			want: "Google Kubernetes Engine(1)",
		},
		{
			name:             "base name and (1) already exist returns suffix (2)",
			inspectionTypeID: "gke",
			existingInspections: []struct {
				id   string
				name string
			}{
				{id: "insp-1", name: "Google Kubernetes Engine"},
				{id: "insp-2", name: "Google Kubernetes Engine(1)"},
			},
			want: "Google Kubernetes Engine(2)",
		},
		{
			name:             "gap in sequence picks lowest available number",
			inspectionTypeID: "gke",
			existingInspections: []struct {
				id   string
				name string
			}{
				{id: "insp-1", name: "Google Kubernetes Engine"},
				{id: "insp-2", name: "Google Kubernetes Engine(2)"},
			},
			want: "Google Kubernetes Engine(1)",
		},
		{
			name:             "excluded inspection ID is ignored",
			inspectionTypeID: "gke",
			existingInspections: []struct {
				id   string
				name string
			}{
				{id: "self-id", name: "Google Kubernetes Engine"},
			},
			excludeID: "self-id",
			want:      "Google Kubernetes Engine",
		},
		{
			name:             "unknown inspection type falls back to Inspection",
			inspectionTypeID: "unknown-type",
			existingInspections: []struct {
				id   string
				name string
			}{
				{id: "insp-1", name: "Inspection"},
			},
			want: "Inspection(1)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(&inspectioncore.IOConfig{})
			if err != nil {
				t.Fatalf("failed to create inspection task server: %v", err)
			}
			if err := server.AddInspectionType(InspectionType{
				Id:   "gke",
				Name: "Google Kubernetes Engine",
			}); err != nil {
				t.Fatalf("failed to add inspection type: %v", err)
			}

			for _, existing := range tc.existingInspections {
				md := typedmap.NewTypedMap()
				typedmap.Set(md, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{
					InspectionName: existing.name,
				})
				runner := &InspectionTaskRunner{
					inspectionServer: server,
					ID:               existing.id,
					metadata:         md.AsReadonly(),
				}
				server.inspections[existing.id] = runner
			}

			got := server.GetDefaultInspectionName(tc.inspectionTypeID, tc.excludeID)
			if got != tc.want {
				t.Errorf("GetDefaultInspectionName(%q, %q) = %q, want %q", tc.inspectionTypeID, tc.excludeID, got, tc.want)
			}
		})
	}
}
