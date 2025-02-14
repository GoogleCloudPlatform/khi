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

package config

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestNewGetConfigResponseFromParameters(t *testing.T) {
	testCases := []struct {
		name       string
		viewerMode *bool
		want       *GetConfigResponse
	}{
		{
			name:       "viewer mode is nil",
			viewerMode: nil,
			want: &GetConfigResponse{
				ViewerMode: false,
			},
		},
		{
			name:       "viewer mode is true",
			viewerMode: testutil.P(true),
			want: &GetConfigResponse{
				ViewerMode: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parameters.Server.ViewerMode = tc.viewerMode
			got := NewGetConfigResponseFromParameters()
			if got.ViewerMode != tc.want.ViewerMode {
				t.Errorf("NewGetConfigResponseFromParameters() = %v, want %v", got, tc.want)
			}
		})

	}
}
