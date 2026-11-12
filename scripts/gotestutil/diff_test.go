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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsTargetGoFile(t *testing.T) {
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
			name:    "empty path excluded",
			relPath: "./",
			want:    false,
		},
		{
			name:    "unit test file excluded",
			relPath: "scripts/mutation-diff/mutator_test.go",
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
			name:    "scripts directory included by default",
			relPath: "scripts/mutation-diff/mutator.go",
			want:    true,
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
			got := IsTargetGoFile(tc.relPath)
			if got != tc.want {
				t.Errorf("IsTargetGoFile(%q) = %v, want %v", tc.relPath, got, tc.want)
			}
		})
	}
}

func TestIsTargetProductionTSFile(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		want    bool
	}{
		{
			name:    "standard component ts file",
			relPath: "web/src/app/app.component.ts",
			want:    true,
		},
		{
			name:    "leading dot-slash ts file",
			relPath: "./web/src/app/services/data.service.ts",
			want:    true,
		},
		{
			name:    "spec test file excluded",
			relPath: "web/src/app/app.component.spec.ts",
			want:    false,
		},
		{
			name:    "storybook stories file excluded",
			relPath: "web/src/app/dialogs/foo.stories.ts",
			want:    false,
		},
		{
			name:    "ambient d.ts file excluded",
			relPath: "web/src/global.d.ts",
			want:    false,
		},
		{
			name:    "mock suffix file excluded",
			relPath: "web/src/app/services/foo.mock.ts",
			want:    false,
		},
		{
			name:    "mock directory excluded",
			relPath: "web/src/app/services/mock/fake-backend.ts",
			want:    false,
		},
		{
			name:    "test helper directory excluded",
			relPath: "web/src/app/common/test/harness.ts",
			want:    false,
		},
		{
			name:    "generated directory excluded",
			relPath: "web/src/app/generated/proto.ts",
			want:    false,
		},
		{
			name:    "environments directory excluded",
			relPath: "web/src/environments/environment.ts",
			want:    false,
		},
		{
			name:    "zzz generated file excluded",
			relPath: "web/src/assets/zzz-fonts.ts",
			want:    false,
		},
		{
			name:    "test bootstrap file excluded",
			relPath: "web/src/test.ts",
			want:    false,
		},
		{
			name:    "outside web/src excluded",
			relPath: "web/karma.conf.ts",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTargetProductionTSFile(tc.relPath)
			if got != tc.want {
				t.Errorf("IsTargetProductionTSFile(%q) = %v, want %v", tc.relPath, got, tc.want)
			}
		})
	}
}

func TestIsTargetTSTestFile(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		want    bool
	}{
		{
			name:    "standard angular spec file",
			relPath: "web/src/app/app.component.spec.ts",
			want:    true,
		},
		{
			name:    "leading dot-slash spec file",
			relPath: "./web/src/app/services/data.service.spec.ts",
			want:    true,
		},
		{
			name:    "production ts file excluded",
			relPath: "web/src/app/app.component.ts",
			want:    false,
		},
		{
			name:    "generated spec excluded",
			relPath: "web/src/app/generated/foo.spec.ts",
			want:    false,
		},
		{
			name:    "zzz spec excluded",
			relPath: "web/src/app/zzz_auto.spec.ts",
			want:    false,
		},
		{
			name:    "outside web/src excluded",
			relPath: "scripts/mutation-diff/sample.spec.ts",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTargetTSTestFile(tc.relPath)
			if got != tc.want {
				t.Errorf("IsTargetTSTestFile(%q) = %v, want %v", tc.relPath, got, tc.want)
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

func TestParseUnifiedDiffSplitCapturesTestLines(t *testing.T) {
	diffText := `diff --git a/pkg/foo/service.go b/pkg/foo/service.go
--- a/pkg/foo/service.go
+++ b/pkg/foo/service.go
@@ -4,0 +5,1 @@
+	doWork()
diff --git a/pkg/foo/service_test.go b/pkg/foo/service_test.go
--- a/pkg/foo/service_test.go
+++ b/pkg/foo/service_test.go
@@ -10,0 +11,2 @@
+func TestService(t *testing.T) {
+}
diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,0 +2,1 @@
+# non-Go file must be ignored by both maps
diff --git a/scripts/mutation-diff/runner_test.go b/scripts/mutation-diff/runner_test.go
--- a/scripts/mutation-diff/runner_test.go
+++ b/scripts/mutation-diff/runner_test.go
@@ -1,0 +2,1 @@
+// scripts test files are included by default
`
	prod, tests, err := ParseUnifiedDiffSplit(strings.NewReader(diffText))
	if err != nil {
		t.Fatalf("ParseUnifiedDiffSplit() unexpected error: %v", err)
	}
	wantProd := map[string]map[int]struct{}{
		"pkg/foo/service.go": {5: {}},
	}
	wantTests := map[string]map[int]struct{}{
		"pkg/foo/service_test.go":              {11: {}, 12: {}},
		"scripts/mutation-diff/runner_test.go": {2: {}},
	}
	if diff := cmp.Diff(wantProd, prod); diff != "" {
		t.Errorf("prod diff mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantTests, tests); diff != "" {
		t.Errorf("test diff mismatch (-want +got):\n%s", diff)
	}
}

func TestParseUnifiedDiffAll(t *testing.T) {
	diffText := `diff --git a/pkg/foo/service.go b/pkg/foo/service.go
--- a/pkg/foo/service.go
+++ b/pkg/foo/service.go
@@ -4,0 +5,1 @@
+	doWork()
diff --git a/pkg/foo/service_test.go b/pkg/foo/service_test.go
--- a/pkg/foo/service_test.go
+++ b/pkg/foo/service_test.go
@@ -10,0 +11,1 @@
+func TestService(t *testing.T) {}
diff --git a/web/src/app/foo.component.ts b/web/src/app/foo.component.ts
--- a/web/src/app/foo.component.ts
+++ b/web/src/app/foo.component.ts
@@ -7,0 +8,2 @@
+  readonly enabled = true;
+  readonly count = 1;
diff --git a/web/src/app/foo.component.spec.ts b/web/src/app/foo.component.spec.ts
--- a/web/src/app/foo.component.spec.ts
+++ b/web/src/app/foo.component.spec.ts
@@ -1,0 +2,2 @@
+it('renders', () => {});
+it('updates', () => {});
`
	got, err := ParseUnifiedDiffAll(strings.NewReader(diffText))
	if err != nil {
		t.Fatalf("ParseUnifiedDiffAll() unexpected error: %v", err)
	}
	want := UnifiedDiffLines{
		Go: map[string]map[int]struct{}{
			"pkg/foo/service.go": {5: {}},
		},
		GoTest: map[string]map[int]struct{}{
			"pkg/foo/service_test.go": {11: {}},
		},
		TS: map[string]map[int]struct{}{
			"web/src/app/foo.component.ts": {8: {}, 9: {}},
		},
		TSTest: map[string]map[int]struct{}{
			"web/src/app/foo.component.spec.ts": {2: {}, 3: {}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseUnifiedDiffAll() mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectChangedLinesAutoFallsBackToGitOnGitRevision(t *testing.T) {
	tmpDir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init", "-b", "main")
	pkgDir := filepath.Join(tmpDir, "pkg", "demo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "demo.go"), []byte("package demo\n"), 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "first")

	if err := os.WriteFile(filepath.Join(pkgDir, "demo.go"), []byte("package demo\n\nfunc Hello() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "second")

	// Create a dummy .jj directory so resolveVCS("auto") prefers jj first, then falls back to git for Git-only rev syntax "HEAD~1".
	if err := os.MkdirAll(filepath.Join(tmpDir, ".jj"), 0o755); err != nil {
		t.Fatalf("mkdir .jj failed: %v", err)
	}

	res, err := CollectChangedLines(context.Background(), tmpDir, DiffOptions{
		VCS:  "auto",
		Base: "HEAD~1",
	})
	if err != nil {
		t.Fatalf("CollectChangedLines() unexpected error: %v", err)
	}
	if res.VCS != "git" {
		t.Errorf("res.VCS = %q, want %q", res.VCS, "git")
	}
	if _, ok := res.ChangedLines["pkg/demo/demo.go"][3]; !ok {
		t.Errorf("expected line 3 of pkg/demo/demo.go in ChangedLines, got: %v", res.ChangedLines)
	}

	// Explicit VCS="jj" must report the jj failure instead of silently falling back to git.
	if _, err := CollectChangedLines(context.Background(), tmpDir, DiffOptions{
		VCS:  "jj",
		Base: "HEAD~1",
	}); err == nil {
		t.Errorf("CollectChangedLines(VCS=jj) expected error on invalid jj repo, got nil")
	}
}
