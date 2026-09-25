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

package inspectioncore

import (
	"errors"
	"testing"
)

// TestInMemoryInspectionNameRegistry_ResolveUniqueName verifies resolving unused and sequentially suffixed names.
func TestInMemoryInspectionNameRegistry_ResolveUniqueName(t *testing.T) {
	testCases := []struct {
		name             string
		setup            func(r *InMemoryInspectionNameRegistry)
		inspectionID     string
		baseName         string
		want             string
		verifyNoMutation func(t *testing.T, r *InMemoryInspectionNameRegistry)
	}{
		{
			name:         "returns baseName when unused",
			setup:        func(r *InMemoryInspectionNameRegistry) {},
			inspectionID: "insp-1",
			baseName:     "GKE Cluster",
			want:         "GKE Cluster",
		},
		{
			name: "returns baseName(1) when baseName is reserved by another inspection",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("other-insp", "GKE Cluster")
			},
			inspectionID: "insp-1",
			baseName:     "GKE Cluster",
			want:         "GKE Cluster(1)",
		},
		{
			name: "returns baseName(2) when baseName and baseName(1) are reserved by other inspections",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("other-1", "GKE Cluster")
				_ = r.ReserveName("other-2", "GKE Cluster(1)")
			},
			inspectionID: "insp-1",
			baseName:     "GKE Cluster",
			want:         "GKE Cluster(2)",
		},
		{
			name: "fills gaps in sequence when baseName and baseName(2) are reserved",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("other-1", "GKE Cluster")
				_ = r.ReserveName("other-2", "GKE Cluster(2)")
			},
			inspectionID: "insp-1",
			baseName:     "GKE Cluster",
			want:         "GKE Cluster(1)",
		},
		{
			name: "returns baseName when reserved by the same inspectionID",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "GKE Cluster")
			},
			inspectionID: "insp-1",
			baseName:     "GKE Cluster",
			want:         "GKE Cluster",
		},
		{
			name:         "falls back to Inspection when baseName is empty",
			setup:        func(r *InMemoryInspectionNameRegistry) {},
			inspectionID: "insp-1",
			baseName:     "",
			want:         "Inspection",
		},
		{
			name:         "falls back to Inspection when baseName is whitespace only",
			setup:        func(r *InMemoryInspectionNameRegistry) {},
			inspectionID: "insp-1",
			baseName:     "   ",
			want:         "Inspection",
		},
		{
			name: "does not mutate existing reservations",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-existing", "My Inspection")
			},
			inspectionID: "insp-new",
			baseName:     "My Inspection",
			want:         "My Inspection(1)",
			verifyNoMutation: func(t *testing.T, r *InMemoryInspectionNameRegistry) {
				if err := r.ReserveName("another-insp", "My Inspection(1)"); err != nil {
					t.Errorf("expected My Inspection(1) to be unreserved, but got error: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewInMemoryInspectionNameRegistry()
			if tc.setup != nil {
				tc.setup(r)
			}
			got := r.ResolveUniqueName(tc.inspectionID, tc.baseName)
			if got != tc.want {
				t.Errorf("ResolveUniqueName() = %q, want %q", got, tc.want)
			}
			if tc.verifyNoMutation != nil {
				tc.verifyNoMutation(t, r)
			}
		})
	}
}

// TestInMemoryInspectionNameRegistry_ReserveName verifies reservation, idempotency, renaming, and conflicts.
func TestInMemoryInspectionNameRegistry_ReserveName(t *testing.T) {
	testCases := []struct {
		name           string
		setup          func(r *InMemoryInspectionNameRegistry)
		inspectionID   string
		reserveName    string
		wantErr        error
		additionalTest func(t *testing.T, r *InMemoryInspectionNameRegistry)
	}{
		{
			name:         "succeeds on unused name",
			setup:        func(r *InMemoryInspectionNameRegistry) {},
			inspectionID: "insp-1",
			reserveName:  "Inspection A",
			wantErr:      nil,
		},
		{
			name: "succeeds idempotently when called again by the same inspectionID with the same name",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Inspection A")
			},
			inspectionID: "insp-1",
			reserveName:  "Inspection A",
			wantErr:      nil,
		},
		{
			name: "releases old name when inspectionID reserves a new name",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Old Name")
			},
			inspectionID: "insp-1",
			reserveName:  "New Name",
			wantErr:      nil,
			additionalTest: func(t *testing.T, r *InMemoryInspectionNameRegistry) {
				if err := r.ReserveName("insp-2", "Old Name"); err != nil {
					t.Errorf("expected Old Name to be released, but got error: %v", err)
				}
			},
		},
		{
			name: "returns ErrInspectionNameEmpty on empty string without releasing previous reservation",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Previous Name")
			},
			inspectionID: "insp-1",
			reserveName:  "",
			wantErr:      ErrInspectionNameEmpty,
			additionalTest: func(t *testing.T, r *InMemoryInspectionNameRegistry) {
				if err := r.ReserveName("insp-2", "Previous Name"); !errors.Is(err, ErrInspectionNameAlreadyInUse) {
					t.Errorf("expected ErrInspectionNameAlreadyInUse, got %v", err)
				}
			},
		},
		{
			name: "returns ErrInspectionNameEmpty on whitespace string without releasing previous reservation",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "Previous Name")
			},
			inspectionID: "insp-1",
			reserveName:  "   ",
			wantErr:      ErrInspectionNameEmpty,
			additionalTest: func(t *testing.T, r *InMemoryInspectionNameRegistry) {
				if err := r.ReserveName("insp-2", "Previous Name"); !errors.Is(err, ErrInspectionNameAlreadyInUse) {
					t.Errorf("expected ErrInspectionNameAlreadyInUse, got %v", err)
				}
			},
		},
		{
			name: "returns ErrInspectionNameAlreadyInUse when reserved by another inspection without releasing previous reservation",
			setup: func(r *InMemoryInspectionNameRegistry) {
				_ = r.ReserveName("insp-1", "My Name")
				_ = r.ReserveName("other-insp", "Taken Name")
			},
			inspectionID: "insp-1",
			reserveName:  "Taken Name",
			wantErr:      ErrInspectionNameAlreadyInUse,
			additionalTest: func(t *testing.T, r *InMemoryInspectionNameRegistry) {
				if err := r.ReserveName("third-insp", "My Name"); !errors.Is(err, ErrInspectionNameAlreadyInUse) {
					t.Errorf("expected My Name to remain reserved by insp-1, got %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewInMemoryInspectionNameRegistry()
			if tc.setup != nil {
				tc.setup(r)
			}
			err := r.ReserveName(tc.inspectionID, tc.reserveName)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ReserveName() error = %v, want %v", err, tc.wantErr)
			}
			if tc.additionalTest != nil {
				tc.additionalTest(t, r)
			}
		})
	}
}
