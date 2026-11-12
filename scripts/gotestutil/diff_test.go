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
			relPath: "scripts/codequality/main_test.go",
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
			relPath: "scripts/codequality/main.go",
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
			relPath: "scripts/codequality/sample.spec.ts",
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

func TestParseUnifiedDiffDetailsAllLanguagesAndDeletions(t *testing.T) {
	testCases := []struct {
		name string
		diff string
		want UnifiedDiffDetails
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
			want: UnifiedDiffDetails{
				ProdLines: map[string]map[int]struct{}{
					"pkg/foo/bar.go": {6: {}, 7: {}, 20: {}},
				},
				TestLines:   map[string]map[int]struct{}{},
				TSLines:     map[string]map[int]struct{}{},
				TSTestLines: map[string]map[int]struct{}{},
			},
		},
		{
			name: "partitions Go prod/test TypeScript prod/spec deleted Go and non-target Go files",
			diff: `diff --git a/pkg/generated/foo.pb.go b/pkg/generated/foo.pb.go
--- a/pkg/generated/foo.pb.go
+++ b/pkg/generated/foo.pb.go
@@ -1,0 +2,1 @@
+	generated := true
diff --git a/pkg/generated/old.pb.go b/pkg/generated/old.pb.go
deleted file mode 100644
--- a/pkg/generated/old.pb.go
+++ /dev/null
@@ -1,1 +0,0 @@
-package generated
diff --git a/pkg/foo/deleted.go b/pkg/foo/deleted.go
deleted file mode 100644
--- a/pkg/foo/deleted.go
+++ /dev/null
@@ -1,1 +0,0 @@
-package foo
diff --git a/pkg/foo/service.go b/pkg/foo/service.go
--- a/pkg/foo/service.go
+++ b/pkg/foo/service.go
@@ -10,3 +10,4 @@ func Run() error {
 	keepBefore()
-	replacedCall()
+	newCallOne()
+	newCallTwo()
 	return nil
diff --git a/pkg/foo/service_test.go b/pkg/foo/service_test.go
--- a/pkg/foo/service_test.go
+++ b/pkg/foo/service_test.go
@@ -1,0 +2,1 @@
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
`,
			want: UnifiedDiffDetails{
				ProdLines: map[string]map[int]struct{}{
					"pkg/foo/service.go": {11: {}, 12: {}},
				},
				TestLines: map[string]map[int]struct{}{
					"pkg/foo/service_test.go": {2: {}},
				},
				TSLines: map[string]map[int]struct{}{
					"web/src/app/foo.component.ts": {8: {}, 9: {}},
				},
				TSTestLines: map[string]map[int]struct{}{
					"web/src/app/foo.component.spec.ts": {2: {}, 3: {}},
				},
				DeletedGoFiles:      []string{"pkg/foo/deleted.go"},
				OtherChangedGoFiles: []string{"pkg/generated/foo.pb.go", "pkg/generated/old.pb.go"},
			},
		},
		{
			name: "resets state cleanly across gitless headers blank hunk lines malformed hunks and diff boundaries",
			diff: `--- a/pkg/foo/stale_old.go
--- /dev/null
+++ /dev/null
+++ b/pkg/foo/a.go
@@ -0,0 +2,3 @@
+	lineTwo()

+	lineFour()
+++ b/pkg/foo/b.go
+ignored_after_new_header_exits_hunk
@@ -0,0 +7,1 @@
+	lineSeven()
+++ b/README.md
@@ -1,0 +1,1 @@
+ignored_non_go_default_target
+++ b/pkg/foo/d.go
@@ -0,0 +3,1 @@
+	lineThree()
--- /dev/null
+ignored_after_old_header_exits_hunk
+++ /dev/null
@@ -1,0 +8,1 @@
+ignored_after_deleted_resets_target
--- a/README.md
+++ /dev/null
--- a/pkg/foo/first_del.go
+++ /dev/null
+++ b/pkg/foo/c.go
@@ -0,0 +9,1 @@
+	lineNine()
@@ malformed hunk @@
+ignored_malformed_hunk
@@ -0,0 +12,1 @@
+	lineTwelve()
diff --git a/binary.dat b/binary.dat
+ignored_after_diff_git_exits_hunk
@@ -1,0 +15,1 @@
+ignored_after_diff_git_resets_target
+++ /dev/null
`,
			want: UnifiedDiffDetails{
				ProdLines: map[string]map[int]struct{}{
					"pkg/foo/a.go": {2: {}, 4: {}},
					"pkg/foo/b.go": {7: {}},
					"pkg/foo/c.go": {9: {}, 12: {}},
					"pkg/foo/d.go": {3: {}},
				},
				TestLines:      map[string]map[int]struct{}{},
				TSLines:        map[string]map[int]struct{}{},
				TSTestLines:    map[string]map[int]struct{}{},
				DeletedGoFiles: []string{"pkg/foo/first_del.go"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseUnifiedDiffDetails(strings.NewReader(tc.diff))
			if err != nil {
				t.Fatalf("ParseUnifiedDiffDetails() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseUnifiedDiffDetails() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectChangedLinesAutoFallsBackToGitOnGitRevision(t *testing.T) {
	tmpDir := t.TempDir()
	initTestGitDiffRepo(t, tmpDir)

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

	baseBytes, exists, err := ReadBaseFile(context.Background(), tmpDir, res, "pkg/demo/demo.go")
	if err != nil || !exists || string(baseBytes) != "package demo\n" {
		t.Errorf("ReadBaseFile(existing) = (%q, %v, %v), want (\"package demo\\n\", true, nil)", string(baseBytes), exists, err)
	}
	if _, missingExists, err := ReadBaseFile(context.Background(), tmpDir, res, "pkg/demo/added_later.go"); err != nil || missingExists {
		t.Errorf("ReadBaseFile(missing) = (%v, %v), want (false, nil)", missingExists, err)
	}

	verifyGitMergeBaseAndTargetDiff(t, tmpDir)
}

func verifyGitMergeBaseAndTargetDiff(t *testing.T, tmpDir string) {
	t.Helper()
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
	// Create a feature branch from HEAD~1 while main has advanced to HEAD so merge-base(main, feat) != main.
	runGit("checkout", "-b", "feat", "HEAD~1")
	featFile := filepath.Join(tmpDir, "pkg", "demo", "feat.go")
	if err := os.WriteFile(featFile, []byte("package demo\n\nfunc Feat() int { return 9 }\n"), 0o600); err != nil {
		t.Fatalf("write feat.go failed: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "feat commit")

	// Default base="" resolves "main" and uses merge-base(main, HEAD), excluding main's second commit on demo.go.
	mbRes, err := CollectChangedLines(context.Background(), tmpDir, DiffOptions{VCS: "git"})
	if err != nil {
		t.Fatalf("CollectChangedLines(default base) error: %v", err)
	}
	if _, hasMainChange := mbRes.ChangedLines["pkg/demo/demo.go"]; hasMainChange {
		t.Errorf("expected merge-base diff to exclude main-only demo.go change, got %v", mbRes.ChangedLines)
	}
	if _, hasFeat := mbRes.ChangedLines["pkg/demo/feat.go"][3]; !hasFeat {
		t.Errorf("expected feat.go line 3 in ChangedLines, got %v", mbRes.ChangedLines)
	}
	mbDemo, exists, err := ReadBaseFile(context.Background(), tmpDir, mbRes, "pkg/demo/demo.go")
	if err != nil || !exists || string(mbDemo) != "package demo\n" {
		t.Errorf("ReadBaseFile(merge-base demo.go) = (%q, %v, %v), want initial content", string(mbDemo), exists, err)
	}

	// Explicit Target uses two-rev diff (base..target).
	targRes, err := CollectChangedLines(context.Background(), tmpDir, DiffOptions{
		VCS:    "git",
		Base:   "HEAD~1",
		Target: "HEAD",
	})
	if err != nil {
		t.Fatalf("CollectChangedLines(Target=HEAD) error: %v", err)
	}
	if _, hasFeat := targRes.ChangedLines["pkg/demo/feat.go"][3]; !hasFeat {
		t.Errorf("expected feat.go line 3 in target diff, got %v", targRes.ChangedLines)
	}
}

func initTestGitDiffRepo(t *testing.T, tmpDir string) {
	t.Helper()
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
}

func TestReadBaseFileValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		res     DiffResult
		relPath string
	}{
		{
			name:    "empty path rejected",
			res:     DiffResult{VCS: "git", ResolvedBaseRev: "HEAD"},
			relPath: "./",
		},
		{
			name:    "missing resolved base rev rejected",
			res:     DiffResult{VCS: "git"},
			relPath: "pkg/a/a.go",
		},
		{
			name:    "unsupported vcs rejected",
			res:     DiffResult{VCS: "svn", ResolvedBaseRev: "r1"},
			relPath: "pkg/a/a.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, exists, err := ReadBaseFile(context.Background(), t.TempDir(), tc.res, tc.relPath); err == nil || exists {
				t.Errorf("ReadBaseFile() = (exists=%v, err=%v), want (false, non-nil)", exists, err)
			}
		})
	}
}

func TestCollectJJDiffAndReadBaseFile(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj binary not installed")
	}
	tmpDir := t.TempDir()
	initTestJJDiffRepo(t, tmpDir)

	testCases := []struct {
		name      string
		base      string
		queryFile string
		wantExist bool
	}{
		{
			name:      "reads existing base file from explicit parent",
			base:      "@-",
			queryFile: "pkg/jjdemo/jjdemo.go",
			wantExist: true,
		},
		{
			name:      "returns false for absent base file using default trunk()",
			base:      "",
			queryFile: "pkg/jjdemo/absent.go",
			wantExist: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := CollectChangedLines(context.Background(), tmpDir, DiffOptions{
				VCS:  "jj",
				Base: tc.base,
			})
			if err != nil {
				t.Fatalf("CollectChangedLines(jj) error: %v", err)
			}
			_, exists, err := ReadBaseFile(context.Background(), tmpDir, res, tc.queryFile)
			if err != nil || exists != tc.wantExist {
				t.Errorf("ReadBaseFile(%s) = (exists=%v, err=%v), want exists=%v", tc.queryFile, exists, err, tc.wantExist)
			}
		})
	}
}

func initTestJJDiffRepo(t *testing.T, tmpDir string) {
	t.Helper()
	runJJ := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), "jj", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(),
			"JJ_CONFIG=/dev/null",
			"JJ_USER=Test",
			"JJ_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("jj %v failed: %v\n%s", args, err, string(out))
		}
	}
	runJJ("git", "init")
	pkgDir := filepath.Join(tmpDir, "pkg", "jjdemo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "jjdemo.go"), []byte("package jjdemo\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runJJ("new", "-m", "second")
	if err := os.WriteFile(filepath.Join(pkgDir, "jjdemo.go"), []byte("package jjdemo\n\nfunc Val() int { return 7 }\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
