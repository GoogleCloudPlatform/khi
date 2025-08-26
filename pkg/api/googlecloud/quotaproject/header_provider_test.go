// Copyright 2024 Google LLC
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

package quotaproject

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGCPQuotaProjectHeaderProvider(t *testing.T) {
	testCases := []struct {
		name         string
		quotaProject string
		wantHeader   string
		wantErr      bool
	}{
		{
			name:         "success",
			quotaProject: "test-project",
			wantHeader:   "test-project",
			wantErr:      false,
		},
		{
			name:         "empty quota project",
			quotaProject: "",
			wantHeader:   "",
			wantErr:      false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHeaderProvider(tt.quotaProject)
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			err = p.AddHeader(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GCPQuotaProjectHeaderProvider.AddHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				gotHeader := req.Header.Get("X-Goog-User-Project")
				if diff := cmp.Diff(tt.wantHeader, gotHeader); diff != "" {
					t.Errorf("GCPQuotaProjectHeaderProvider.AddHeader() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
