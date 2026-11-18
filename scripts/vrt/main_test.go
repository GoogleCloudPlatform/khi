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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/gcs"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/report"
)

type fakeStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: map[string][]byte{}}
}

func (f *fakeStore) Get(_ context.Context, object string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.data[object]
	if !ok {
		return nil, gcs.ErrNotFound
	}
	return b, nil
}

func (f *fakeStore) Put(_ context.Context, object string, _ string, _ string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.data[object] = cp
	return nil
}

func (f *fakeStore) Exists(_ context.Context, object string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.data[object]
	return ok, nil
}

func writeFixtureSnapshot(t *testing.T, dir string, sha string, c color.NRGBA) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 12, 12))
	for y := range 12 {
		for x := range 12 {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	rel := "images/story/header--default.png"
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(abs, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write png failed: %v", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	m := capture.Manifest{
		Version:            capture.ManifestVersion,
		CommitSHA:          sha,
		BuilderFingerprint: "chrome-136-trixie",
		Entries: map[string]capture.SnapshotEntry{
			"story/header--default": {
				ID:           "story/header--default",
				StoryID:      "header--default",
				Title:        "Header",
				Name:         "Default",
				ImageRelPath: rel,
				SHA256:       hex.EncodeToString(sum[:]),
				Width:        12,
				Height:       12,
			},
		},
	}
	if err := capture.WriteManifest(filepath.Join(dir, capture.ManifestFileName), m); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}
}

func TestRunCLI_CompareAndGCSModes(t *testing.T) {
	baseDir := t.TempDir()
	headDir := t.TempDir()
	reportDir := t.TempDir()

	writeFixtureSnapshot(t, baseDir, "base11111111", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	writeFixtureSnapshot(t, headDir, "head22222222", color.NRGBA{R: 220, G: 20, B: 20, A: 255})

	store := newFakeStore()
	var stdout, stderr bytes.Buffer
	deps := cliDeps{
		stdout: &stdout,
		stderr: &stderr,
		fingerprintFn: func(_ context.Context) (string, error) {
			return "chrome-136-trixie", nil
		},
		newGCSClientFn: func(_ context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error) {
			return gcs.NewClient(store, bucket, mode, workers), nil
		},
	}

	// 1. Fingerprint mode.
	code := runCLI(context.Background(), []string{"--mode=fingerprint"}, deps)
	if code != 0 || strings.TrimSpace(stdout.String()) != "chrome-136-trixie" {
		t.Fatalf("fingerprint mode = (code %d, out %q, err %q), want (0, chrome-136-trixie)", code, stdout.String(), stderr.String())
	}

	// 2. Compare mode with GCS upload + comment envelope + advisory (fail-on-diff=false).
	stdout.Reset()
	stderr.Reset()
	code = runCLI(context.Background(), []string{
		"--mode=compare",
		"--base-dir=" + baseDir,
		"--head-dir=" + headDir,
		"--report-dir=" + reportDir,
		"--gcs-bucket=khi-vrt-artifacts",
		"--gcs-access-mode=console",
		"--commit-sha=head22222222",
		"--base-sha=base11111111",
		"--base-branch=main",
		"--run-id=build-42",
		"--upload-head-snapshot",
		"--upload-report",
		"--comment-envelope",
		"--include-aa",
	}, deps)
	if code != 0 {
		t.Fatalf("compare advisory mode exit code = %d, want 0 (stderr=%s)", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "GCS HTML report:") {
		t.Errorf("compare stdout missing GCS HTML report line:\n%s", stdout.String())
	}
	decoded, found, err := report.ExtractCommentEnvelope(stdout.String())
	if err != nil || !found {
		t.Fatalf("ExtractCommentEnvelope() = (found=%v, err=%v), want found=true", found, err)
	}
	if !strings.Contains(decoded, "https://storage.cloud.google.com/khi-vrt-artifacts/reports/head22222222/build-42/index.html") {
		t.Errorf("comment envelope missing expected console GCS report URL:\n%s", decoded)
	}
	uploadedHTML, err := store.Get(context.Background(), "reports/head22222222/build-42/index.html")
	if err != nil {
		t.Fatalf("expected uploaded index.html in GCS store: %v", err)
	}
	wantAssetBase := `"assetBaseUrl":"https://storage.cloud.google.com/khi-vrt-artifacts/reports/head22222222/build-42"`
	if !strings.Contains(string(uploadedHTML), wantAssetBase) {
		t.Errorf("uploaded index.html missing %q", wantAssetBase)
	}

	// 3. Verify has-snapshot now hits for uploaded HEAD snapshot and misses for unknown SHA.
	code = runCLI(context.Background(), []string{
		"--mode=has-snapshot",
		"--gcs-bucket=khi-vrt-artifacts",
		"--fingerprint=chrome-136-trixie",
		"--commit-sha=head22222222",
	}, deps)
	if code != 0 {
		t.Errorf("has-snapshot(hit) exit code = %d, want 0", code)
	}
	code = runCLI(context.Background(), []string{
		"--mode=has-snapshot",
		"--gcs-bucket=khi-vrt-artifacts",
		"--fingerprint=chrome-136-trixie",
		"--commit-sha=nonexistent",
	}, deps)
	if code != exitCodeCacheMiss {
		t.Errorf("has-snapshot(miss) exit code = %d, want %d", code, exitCodeCacheMiss)
	}

	// 4. Verify fail-on-diff exits 1 when visual diffs exist.
	stdout.Reset()
	code = runCLI(context.Background(), []string{
		"--mode=compare",
		"--base-dir=" + baseDir,
		"--head-dir=" + headDir,
		"--report-dir=" + t.TempDir(),
		"--include-aa",
		"--fail-on-diff",
	}, deps)
	if code != 1 {
		t.Errorf("compare --fail-on-diff exit code = %d, want 1", code)
	}
	if strings.Contains(stdout.String(), "GCS HTML report:") {
		t.Errorf("local-only compare unexpectedly printed GCS HTML report:\n%s", stdout.String())
	}
}

func TestRunCLI_SkipCommentMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := cliDeps{stdout: &stdout, stderr: &stderr}

	code := runCLI(context.Background(), []string{
		"--mode=skip-comment",
		"--commit-sha=abc123456789",
		"--base-sha=def987654321",
		"--base-branch=main",
		"--comment-envelope",
	}, deps)
	if code != 0 {
		t.Fatalf("skip-comment exit code = %d, want 0 (stderr=%s)", code, stderr.String())
	}

	decoded, found, err := report.ExtractCommentEnvelope(stdout.String())
	if err != nil || !found {
		t.Fatalf("ExtractCommentEnvelope() = (found=%v, err=%v), want found=true", found, err)
	}
	if !strings.Contains(decoded, "SKIPPED") {
		t.Errorf("expected SKIPPED in decoded comment body:\n%s", decoded)
	}

	// Plain skip-comment (without --comment-envelope) writes raw Markdown directly.
	stdout.Reset()
	code = runCLI(context.Background(), []string{"--mode=skip-comment", "--commit-sha=abc123456789"}, deps)
	if code != 0 {
		t.Fatalf("skip-comment plain exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), report.PRCommentMarker) || strings.Contains(stdout.String(), report.EnvelopeBeginMarker) {
		t.Errorf("plain skip-comment stdout unexpected:\n%s", stdout.String())
	}
}

func TestRunCLI_CaptureDownloadAndValidationModes(t *testing.T) {
	store := newFakeStore()
	capDir := t.TempDir()
	writeFixtureSnapshot(t, capDir, "sha-cap", color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	capManifest, err := capture.ReadManifest(filepath.Join(capDir, capture.ManifestFileName))
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}

	var capturedCfg capture.Config
	var stdout, stderr bytes.Buffer
	deps := cliDeps{
		stdout: &stdout,
		stderr: &stderr,
		captureFn: func(_ context.Context, cfg capture.Config) (capture.Manifest, error) {
			capturedCfg = cfg
			return capManifest, nil
		},
		newGCSClientFn: func(_ context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error) {
			return gcs.NewClient(store, bucket, mode, workers), nil
		},
	}

	// 1. Capture mode (uppercase mode flag verifies normalization) + upload snapshot to GCS.
	code := runCLI(context.Background(), []string{
		"--mode=  CAPTURE ",
		"--out-dir=" + capDir,
		"--gcs-bucket=khi-vrt",
		"--commit-sha=sha-cap",
		"--upload-head-snapshot",
	}, deps)
	if code != 0 {
		t.Fatalf("capture mode exit = %d, want 0 (stderr=%s)", code, stderr.String())
	}
	if capturedCfg.OutputDir != capDir || capturedCfg.CommitSHA != "sha-cap" {
		t.Errorf("capturedCfg mismatch: %+v", capturedCfg)
	}

	// 2. Download-snapshot hit and miss + validation errors.
	dlDir := t.TempDir()
	code = runCLI(context.Background(), []string{
		"--mode=download-snapshot",
		"--gcs-bucket=khi-vrt",
		"--fingerprint=chrome-136-trixie",
		"--commit-sha=sha-cap",
		"--out-dir=" + dlDir,
	}, deps)
	if code != 0 {
		t.Fatalf("download-snapshot(hit) exit = %d, want 0 (stderr=%s)", code, stderr.String())
	}
	code = runCLI(context.Background(), []string{
		"--mode=download-snapshot",
		"--gcs-bucket=khi-vrt",
		"--fingerprint=chrome-136-trixie",
		"--commit-sha=missing-sha",
		"--out-dir=" + t.TempDir(),
	}, deps)
	if code != exitCodeCacheMiss {
		t.Errorf("download-snapshot(miss) exit = %d, want %d", code, exitCodeCacheMiss)
	}

	// 3. Compare mode auto-captures when head manifest does not yet exist on disk.
	freshHeadDir := t.TempDir()
	deps.captureFn = func(_ context.Context, cfg capture.Config) (capture.Manifest, error) {
		writeFixtureSnapshot(t, cfg.OutputDir, "sha-auto", color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		return capture.ReadManifest(filepath.Join(cfg.OutputDir, capture.ManifestFileName))
	}
	code = runCLI(context.Background(), []string{
		"--mode=compare",
		"--base-dir=" + capDir,
		"--head-dir=" + freshHeadDir,
		"--report-dir=" + t.TempDir(),
	}, deps)
	if code != 0 {
		t.Fatalf("compare auto-capture exit = %d, want 0 (stderr=%s)", code, stderr.String())
	}

	// 4. Capture mode returns 1 when captured manifest contains story errors.
	deps.captureFn = func(_ context.Context, _ capture.Config) (capture.Manifest, error) {
		return capture.Manifest{
			Entries: map[string]capture.SnapshotEntry{
				"story/fail": {ID: "story/fail", Error: "boom"},
			},
		}, nil
	}
	if code := runCLI(context.Background(), []string{"--mode=capture", "--out-dir=" + t.TempDir()}, deps); code != 1 {
		t.Errorf("capture(with errors) exit = %d, want 1", code)
	}

	// 5. Validation and error branches.
	errorArgs := [][]string{
		{"--mode=bogus"},
		{"--mode=has-snapshot"},
		{"--mode=has-snapshot", "--gcs-bucket=b", "--fingerprint=fp"},
		{"--mode=has-snapshot", "--gcs-bucket=b", "--commit-sha=sha"},
		{"--mode=download-snapshot", "--gcs-bucket=b"},
		{"--mode=download-snapshot", "--gcs-bucket=b", "--fingerprint=fp"},
		{"--mode=download-snapshot", "--gcs-bucket=b", "--commit-sha=sha"},
	}
	for _, args := range errorArgs {
		if got := runCLI(context.Background(), args, deps); got != 1 {
			t.Errorf("runCLI(%v) = %d, want 1", args, got)
		}
	}

	// 6. Verify compareExitCode returns 1 on errored reports and countCaptureErrors counts failed entries.
	if got := compareExitCode(report.ComparisonReport{Summary: report.Summary{Errored: 1}}, false); got != 1 {
		t.Errorf("compareExitCode(errored) = %d, want 1", got)
	}
	if got := countCaptureErrors(capture.Manifest{Entries: map[string]capture.SnapshotEntry{"a": {Error: "e"}}}); got != 1 {
		t.Errorf("countCaptureErrors() = %d, want 1", got)
	}
}

func TestUploadGuards_CaptureAndCompare(t *testing.T) {
	capDir := t.TempDir()
	writeFixtureSnapshot(t, capDir, "sha-guard", color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	okManifest, err := capture.ReadManifest(filepath.Join(capDir, capture.ManifestFileName))
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	errManifest := capture.Manifest{
		BuilderFingerprint: "chrome-136-trixie",
		Entries:            map[string]capture.SnapshotEntry{"story/fail": {ID: "story/fail", Error: "boom"}},
	}

	// Capture upload guards: must NOT upload when flag is false, commitSHA is empty, or capture has errors.
	captureGuardCases := []struct {
		name     string
		args     []string
		manifest capture.Manifest
		wantCode int
	}{
		{
			name:     "upload flag disabled",
			args:     []string{"--mode=capture", "--out-dir=" + capDir, "--gcs-bucket=b", "--commit-sha=sha-guard"},
			manifest: okManifest,
			wantCode: 0,
		},
		{
			name:     "missing commit sha",
			args:     []string{"--mode=capture", "--out-dir=" + capDir, "--gcs-bucket=b", "--upload-head-snapshot"},
			manifest: okManifest,
			wantCode: 0,
		},
		{
			name:     "capture errors block snapshot upload",
			args:     []string{"--mode=capture", "--out-dir=" + capDir, "--gcs-bucket=b", "--commit-sha=sha-guard", "--upload-head-snapshot"},
			manifest: errManifest,
			wantCode: 1,
		},
	}
	for _, tc := range captureGuardCases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeStore()
			var stdout, stderr bytes.Buffer
			deps := cliDeps{
				stdout: &stdout,
				stderr: &stderr,
				captureFn: func(_ context.Context, _ capture.Config) (capture.Manifest, error) {
					return tc.manifest, nil
				},
				newGCSClientFn: func(_ context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error) {
					return gcs.NewClient(store, bucket, mode, workers), nil
				},
			}
			if got := runCLI(context.Background(), tc.args, deps); got != tc.wantCode {
				t.Errorf("runCLI() = %d, want %d", got, tc.wantCode)
			}
			if len(store.data) != 0 {
				t.Errorf("expected zero uploaded objects, got %d", len(store.data))
			}
		})
	}

	// prepareCompareUpload branches.
	store := newFakeStore()
	deps := cliDeps{
		newGCSClientFn: func(_ context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error) {
			return gcs.NewClient(store, bucket, mode, workers), nil
		},
	}
	planNone, err := prepareCompareUpload(context.Background(), cliFlags{gcsBucket: "b", gcsAccessMode: "console"}, deps, okManifest)
	if err != nil || planNone.client != nil {
		t.Fatalf("prepareCompareUpload(no upload flags) = (client=%v, err=%v), want (nil, nil)", planNone.client, err)
	}
	planHeadOnly, err := prepareCompareUpload(context.Background(), cliFlags{
		gcsBucket: "b", gcsAccessMode: "console", uploadHeadSnapshot: true,
	}, deps, okManifest)
	if err != nil || planHeadOnly.client == nil || planHeadOnly.reportBaseURL != "" || planHeadOnly.inlineImages {
		t.Fatalf("prepareCompareUpload(head only, console) unexpected: %+v, err=%v", planHeadOnly, err)
	}
	planReportPublic, err := prepareCompareUpload(context.Background(), cliFlags{
		gcsBucket: "b", gcsAccessMode: "public", uploadReport: true, runID: "r1",
	}, deps, okManifest)
	if err != nil || planReportPublic.client == nil || planReportPublic.reportBaseURL == "" || !planReportPublic.inlineImages {
		t.Fatalf("prepareCompareUpload(report only, public) unexpected: %+v, err=%v", planReportPublic, err)
	}

	// executeCompareUpload guards: must NOT upload snapshot or report when flags/metadata/errors forbid it.
	reportDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(reportDir, "index.html"), []byte("<html/>"), 0o600)
	client := gcs.NewClient(store, "b", gcs.AccessModeConsole, 1)

	execCases := []struct {
		name     string
		plan     compareUploadPlan
		manifest capture.Manifest
	}{
		{
			name:     "both upload flags false",
			plan:     compareUploadPlan{client: client, headSHA: "sha", fingerprint: "fp"},
			manifest: okManifest,
		},
		{
			name:     "head snapshot skipped when headSHA empty",
			plan:     compareUploadPlan{client: client, uploadHeadSnapshot: true, fingerprint: "fp"},
			manifest: okManifest,
		},
		{
			name:     "head snapshot skipped when manifest has errors",
			plan:     compareUploadPlan{client: client, uploadHeadSnapshot: true, headSHA: "sha", fingerprint: "fp"},
			manifest: errManifest,
		},
	}
	for _, tc := range execCases {
		t.Run(tc.name, func(t *testing.T) {
			store.data = map[string][]byte{}
			if err := executeCompareUpload(context.Background(), tc.plan, capDir, tc.manifest, reportDir); err != nil {
				t.Fatalf("executeCompareUpload() unexpected error = %v", err)
			}
			if len(store.data) != 0 {
				t.Errorf("expected zero uploaded objects, got %v", store.data)
			}
		})
	}
}
