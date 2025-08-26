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

package index

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
)

func TestBaseTagGenerator(t *testing.T) {
	testCases := []struct {
		name       string
		before     func()
		after      func()
		wantResult string
	}{
		{
			name: "With frontend resource base path env",
			before: func() {
				parameters.Server.FrontendResourceBasePath = testutil.P("/foo/bar/")
			},
			after: func() {
				parameters.Server.FrontendResourceBasePath = nil
			},
			wantResult: `<base href="/foo/bar/">`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			defer tc.after()
			got := (&BaseTagGenerator{}).GenerateTags()
			if len(got) != 1 {
				t.Errorf("len(GenerateTags())=%d, want 1", len(got))
			}
			if got[0] != tc.wantResult {
				t.Errorf("GenerateTags()[0]=%s, want %s", got, tc.wantResult)
			}
		})
	}
}

func TestServerBaseMetaTagGenerator(t *testing.T) {
	testCases := []struct {
		name       string
		before     func()
		after      func()
		wantResult string
	}{
		{
			name: "With server base path env",
			before: func() {
				parameters.Server.BasePath = testutil.P("/foo/bar/")
			},
			after: func() {
				parameters.Server.BasePath = nil
			},
			wantResult: `<meta id="server-base-path" content="/foo/bar/">`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			defer tc.after()
			got := (&ServerBaseMetaTagGenerator{}).GenerateTags()
			if len(got) != 1 {
				t.Errorf("len(GenerateTags())=%d, want 1", len(got))
			}
			if got[0] != tc.wantResult {
				t.Errorf("GenerateTags()[0]=%s, want %s", got, tc.wantResult)
			}
		})
	}
}
