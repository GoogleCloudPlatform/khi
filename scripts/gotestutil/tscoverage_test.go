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

package gotestutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseIstanbulCoverageJSONAndCoveredLineSet(t *testing.T) {
	payload := `{
  "/workspace/khi/web/src/app/utils/math-util.ts": {
    "path": "/workspace/khi/web/src/app/utils/math-util.ts",
    "statementMap": {
      "0": { "start": { "line": 4, "column": 2 }, "end": { "line": 8, "column": 3 } },
      "1": { "start": { "line": 5, "column": 4 }, "end": { "line": 5, "column": 15 } },
      "2": { "start": { "line": 7, "column": 4 }, "end": { "line": 7, "column": 15 } }
    },
    "fnMap": {
      "0": {
        "name": "clamp",
        "decl": { "start": { "line": 3, "column": 16 }, "end": { "line": 3, "column": 21 } },
        "loc": { "start": { "line": 3, "column": 42 }, "end": { "line": 9, "column": 1 } }
      }
    },
    "branchMap": {
      "0": {
        "type": "cond-expr",
        "loc": { "start": { "line": 10, "column": 9 }, "end": { "line": 12, "column": 14 } },
        "locations": [
          { "start": { "line": 11, "column": 6 }, "end": { "line": 11, "column": null } },
          { "start": { "line": 12, "column": 6 }, "end": { "line": 12, "column": 14 } }
        ]
      }
    },
    "s": { "0": 2, "1": 0, "2": 2 },
    "f": { "0": 2 },
    "b": { "0": [2, 0] }
  },
  "/workspace/khi/web/src/app/utils/math-util.spec.ts": {
    "path": "/workspace/khi/web/src/app/utils/math-util.spec.ts",
    "statementMap": {
      "0": { "start": { "line": 1, "column": 0 }, "end": { "line": 1, "column": 10 } }
    },
    "fnMap": {},
    "branchMap": {},
    "s": { "0": 1 },
    "f": {},
    "b": {}
  }
}`

	got, err := ParseIstanbulCoverageJSON(strings.NewReader(payload), "/workspace/khi")
	if err != nil {
		t.Fatalf("ParseIstanbulCoverageJSON() unexpected error: %v", err)
	}

	fc, ok := got["web/src/app/utils/math-util.ts"]
	if !ok {
		t.Fatalf("expected coverage entry for web/src/app/utils/math-util.ts")
	}

	gotCovered := CoveredTSLineSet(fc)
	wantCovered := map[int]struct{}{
		3:  {},
		4:  {},
		6:  {},
		7:  {},
		8:  {},
		9:  {},
		11: {},
	}
	if diff := cmp.Diff(wantCovered, gotCovered); diff != "" {
		t.Errorf("CoveredTSLineSet() mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadTSCoverageIfPresent(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "missing.json")
	gotMissing, err := LoadTSCoverageIfPresent(missingPath, tmpDir)
	if err != nil {
		t.Fatalf("LoadTSCoverageIfPresent(missing) unexpected error: %v", err)
	}
	if len(gotMissing) != 0 {
		t.Errorf("len(gotMissing) = %d, want 0", len(gotMissing))
	}

	validPath := filepath.Join(tmpDir, "coverage-final.json")
	if err := os.WriteFile(validPath, []byte(`{"src/app/foo.ts":{"path":"","statementMap":{"0":{"start":{"line":2,"column":0},"end":{"line":2,"column":8}}},"fnMap":{},"branchMap":{},"s":{"0":1},"f":{},"b":{}}}`), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	gotValid, err := LoadTSCoverageIfPresent(validPath, tmpDir)
	if err != nil {
		t.Fatalf("LoadTSCoverageIfPresent(valid) unexpected error: %v", err)
	}
	if _, ok := gotValid["web/src/app/foo.ts"]; !ok {
		t.Errorf("expected normalized key web/src/app/foo.ts in %+v", gotValid)
	}
}
