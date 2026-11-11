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

package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsTargetProductionGoFile(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		want    bool
	}{
		{
			name:    "standard pkg file",
			relPath: "pkg/common/set/set.go",
			want:    true,
		},
		{
			name:    "cmd entrypoint file",
			relPath: "./cmd/kubernetes-history-inspector/main.go",
			want:    true,
		},
		{
			name:    "unit test file excluded",
			relPath: "pkg/common/set/set_test.go",
			want:    false,
		},
		{
			name:    "generated protobuf directory excluded",
			relPath: "pkg/generated/khi/v1/service.pb.go",
			want:    false,
		},
		{
			name:    "zzz generated file excluded",
			relPath: "pkg/task/inspection/zzz_register.go",
			want:    false,
		},
		{
			name:    "testutil package excluded",
			relPath: "pkg/testutil/golden.go",
			want:    false,
		},
		{
			name:    "scripts directory excluded",
			relPath: "scripts/backend-codegen/main.go",
			want:    false,
		},
		{
			name:    "go file under web node_modules excluded",
			relPath: "web/node_modules/flatted/golang/pkg/flatted/flatted.go",
			want:    false,
		},
		{
			name:    "frontend typescript file excluded",
			relPath: "web/src/app/app.component.ts",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTargetProductionGoFile(tc.relPath)
			if got != tc.want {
				t.Errorf("IsTargetProductionGoFile(%q) = %v, want %v", tc.relPath, got, tc.want)
			}
		})
	}
}

func TestParseUnifiedDiff(t *testing.T) {
	testCases := []struct {
		name string
		diff string
		want map[string]map[int]struct{}
	}{
		{
			name: "zero context additions modifications and pure deletions",
			diff: `diff --git a/pkg/foo/bar.go b/pkg/foo/bar.go
index 1111111..2222222 100644
--- a/pkg/foo/bar.go
+++ b/pkg/foo/bar.go
@@ -5,0 +6,2 @@ func Example() {
+	a := 1
+	b := 2
@@ -12,2 +14,0 @@ func Example() {
-	old1()
-	old2()
@@ -20 +20 @@ func Example() {
-	return a
+	return a + b
`,
			want: map[string]map[int]struct{}{
				"pkg/foo/bar.go": {
					6:  {},
					7:  {},
					20: {},
				},
			},
		},
		{
			name: "diff with surrounding context lines and excluded files",
			diff: `diff --git a/pkg/generated/foo.pb.go b/pkg/generated/foo.pb.go
--- a/pkg/generated/foo.pb.go
+++ b/pkg/generated/foo.pb.go
@@ -1,0 +2,2 @@
+	generated := true
+	_ = generated
diff --git a/pkg/foo/bar_test.go b/pkg/foo/bar_test.go
--- a/pkg/foo/bar_test.go
+++ b/pkg/foo/bar_test.go
@@ -1,0 +2,1 @@
+func TestBar(t *testing.T) {}
diff --git a/pkg/foo/deleted.go b/pkg/foo/deleted.go
deleted file mode 100644
--- a/pkg/foo/deleted.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package foo
-
-func Gone() {}
diff --git a/pkg/foo/service.go b/pkg/foo/service.go
--- a/pkg/foo/service.go
+++ b/pkg/foo/service.go
@@ -10,4 +10,5 @@ func Run() error {
 	keepBefore()
-	replacedCall()
+	newCallOne()
+	newCallTwo()
 	keepAfter()
 	return nil
`,
			want: map[string]map[int]struct{}{
				"pkg/foo/service.go": {
					11: {},
					12: {},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseUnifiedDiff(strings.NewReader(tc.diff))
			if err != nil {
				t.Fatalf("ParseUnifiedDiff() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseUnifiedDiff() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
