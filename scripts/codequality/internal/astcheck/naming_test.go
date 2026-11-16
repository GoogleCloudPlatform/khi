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

package astcheck

import "testing"

func TestCheckCatchAllFilename(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		wantBad bool
		wantMsg string
	}{
		{
			name:    "domain named file allowed",
			relPath: "pkg/log/parser.go",
			wantBad: false,
			wantMsg: "",
		},
		{
			name:    "non-go file ignored",
			relPath: "docs/util.md",
			wantBad: false,
			wantMsg: "",
		},
		{
			name:    "exact helpers_test.go rejected",
			relPath: "pkg/log/helpers_test.go",
			wantBad: true,
			wantMsg: `file "helpers_test.go" uses catch-all name "helpers.go"; name files after their domain responsibility`,
		},
		{
			name:    "suffix string_utils.go rejected",
			relPath: "pkg/log/string_utils.go",
			wantBad: true,
			wantMsg: `file "string_utils.go" uses catch-all suffix "_utils.go"; group code into domain-named files`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg, bad := CheckCatchAllFilename(tc.relPath)
			if bad != tc.wantBad {
				t.Errorf("CheckCatchAllFilename(%q) bad = %v, want %v", tc.relPath, bad, tc.wantBad)
			}
			if msg != tc.wantMsg {
				t.Errorf("CheckCatchAllFilename(%q) msg = %q, want %q", tc.relPath, msg, tc.wantMsg)
			}
		})
	}
}
