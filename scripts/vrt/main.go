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
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/gcs"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/imagediff"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/report"
)

type cliDeps struct {
	stdout         io.Writer
	stderr         io.Writer
	fingerprintFn  func(ctx context.Context) (string, error)
	captureFn      func(ctx context.Context, cfg capture.Config) (capture.Manifest, error)
	newGCSClientFn func(ctx context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error)
}

type compareUploadPlan struct {
	client             *gcs.Client
	headSHA            string
	fingerprint        string
	uploadHeadSnapshot bool
	uploadReport       bool
	reportPrefix       string
	reportBaseURL      string
	inlineImages       bool
}

func main() {
	deps := cliDeps{
		stdout:        os.Stdout,
		stderr:        os.Stderr,
		fingerprintFn: capture.Fingerprint,
		captureFn:     capture.Run,
		newGCSClientFn: func(ctx context.Context, bucket string, mode gcs.AccessMode, workers int) (*gcs.Client, error) {
			store, err := gcs.NewStorageStore(ctx, bucket)
			if err != nil {
				return nil, err
			}
			return gcs.NewClient(store, bucket, mode, workers), nil
		},
	}
	code := runCLI(context.Background(), os.Args[1:], deps)
	os.Exit(code)
}

func runCLI(ctx context.Context, args []string, deps cliDeps) int {
	flags, err := parseFlags(args, deps.stderr)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	switch flags.mode {
	case modeFingerprint:
		return runFingerprint(ctx, deps)
	case modeHasSnapshot:
		return runHasSnapshot(ctx, flags, deps)
	case modeDownloadSnapshot:
		return runDownloadSnapshot(ctx, flags, deps)
	case modeCapture:
		return runCaptureMode(ctx, flags, deps)
	case modeCompare:
		return runCompareMode(ctx, flags, deps)
	default:
		return runSkipCommentMode(flags, deps)
	}
}

func runFingerprint(ctx context.Context, deps cliDeps) int {
	fp, err := deps.fingerprintFn(ctx)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(deps.stdout, fp)
	return 0
}

func runHasSnapshot(ctx context.Context, f cliFlags, deps cliDeps) int {
	client, err := buildGCSClient(ctx, f, deps)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	if f.fingerprint == "" || f.commitSHA == "" {
		fmt.Fprintln(deps.stderr, "error: --fingerprint and --commit-sha are required for has-snapshot")
		return 1
	}
	exists, err := client.HasSnapshot(ctx, f.fingerprint, f.commitSHA)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	if !exists {
		return exitCodeCacheMiss
	}
	return 0
}

func runDownloadSnapshot(ctx context.Context, f cliFlags, deps cliDeps) int {
	client, err := buildGCSClient(ctx, f, deps)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	if f.fingerprint == "" || f.commitSHA == "" {
		fmt.Fprintln(deps.stderr, "error: --fingerprint and --commit-sha are required for download-snapshot")
		return 1
	}
	manifest, err := client.DownloadSnapshot(ctx, f.fingerprint, f.commitSHA, f.outDir)
	if err != nil {
		if errors.Is(err, gcs.ErrNotFound) {
			return exitCodeCacheMiss
		}
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintf(deps.stdout, "Downloaded %d snapshot entries to %s\n", len(manifest.Entries), f.outDir)
	return 0
}

func runCaptureMode(ctx context.Context, f cliFlags, deps cliDeps) int {
	manifest, err := deps.captureFn(ctx, buildCaptureConfig(f, f.outDir, deps.stderr))
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}
	errCount := countCaptureErrors(manifest)
	fmt.Fprintf(deps.stdout, "Captured %d stories into %s (fingerprint=%s, errors=%d)\n",
		len(manifest.Entries), f.outDir, manifest.BuilderFingerprint, errCount)

	if f.uploadHeadSnapshot && f.gcsBucket != "" && f.commitSHA != "" && errCount == 0 {
		client, err := buildGCSClient(ctx, f, deps)
		if err != nil {
			fmt.Fprintf(deps.stderr, "error: %v\n", err)
			return 1
		}
		fp := f.fingerprint
		if fp == "" {
			fp = manifest.BuilderFingerprint
		}
		if err := client.UploadSnapshot(ctx, fp, f.commitSHA, f.outDir, manifest); err != nil {
			fmt.Fprintf(deps.stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(deps.stdout, "Uploaded snapshot to gs://%s/%s\n", f.gcsBucket, gcs.SnapshotPrefix(fp, f.commitSHA))
	}

	if errCount > 0 {
		return 1
	}
	return 0
}

func runCompareMode(ctx context.Context, f cliFlags, deps cliDeps) int {
	headDir := cmp.Or(f.headDir, f.outDir)
	headManifest, err := loadOrCaptureHeadManifest(ctx, f, headDir, deps)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	uploadPlan, err := prepareCompareUpload(ctx, f, deps, headManifest)
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	rep, err := report.CompareDirectories(report.CompareConfig{
		BaseDir:          f.baseDir,
		HeadDir:          headDir,
		ReportDir:        f.reportDir,
		Workers:          f.workers,
		BaselineCacheHit: f.baselineCacheHit,
		AssetBaseURL:     uploadPlan.reportBaseURL,
		DiffOptions: imagediff.Options{
			ColorThreshold:      f.colorThreshold,
			IncludeAntialiasing: f.includeAA,
			MaxDiffPixelRatio:   f.maxDiffPixelRatio,
		},
	})
	if err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	if err := executeCompareUpload(ctx, uploadPlan, headDir, headManifest, f.reportDir); err != nil {
		fmt.Fprintf(deps.stderr, "error: %v\n", err)
		return 1
	}

	printCompareSummary(deps.stdout, rep, f.reportDir, uploadPlan.reportBaseURL)
	emitCompareCommentEnvelope(f, deps.stdout, rep, uploadPlan.reportBaseURL, uploadPlan.inlineImages)
	return compareExitCode(rep, f.failOnDiff)
}

func loadOrCaptureHeadManifest(ctx context.Context, f cliFlags, headDir string, deps cliDeps) (capture.Manifest, error) {
	manifestPath := filepath.Join(headDir, capture.ManifestFileName)
	if _, statErr := os.Stat(manifestPath); errors.Is(statErr, os.ErrNotExist) {
		return deps.captureFn(ctx, buildCaptureConfig(f, headDir, deps.stderr))
	}
	return capture.ReadManifest(manifestPath)
}

func emitCompareCommentEnvelope(f cliFlags, w io.Writer, rep report.ComparisonReport, reportBaseURL string, inlineImages bool) {
	if !f.commentEnvelope {
		return
	}
	body := report.FormatPRCommentBody(rep, report.PRCommentMeta{
		CommitSHA:     cmp.Or(f.commitSHA, rep.HeadSHA),
		BaseSHA:       cmp.Or(f.baseSHA, rep.BaseSHA),
		BaseBranch:    f.baseBranch,
		BuildURL:      f.buildURL,
		ReportBaseURL: reportBaseURL,
		InlineImages:  inlineImages,
	})
	report.WriteCommentEnvelope(w, body)
}

func compareExitCode(rep report.ComparisonReport, failOnDiff bool) int {
	if rep.HasErrors() {
		return 1
	}
	if failOnDiff && rep.HasVisualDiffs() {
		return 1
	}
	return 0
}

func runSkipCommentMode(f cliFlags, deps cliDeps) int {
	reason := f.skipReason
	if reason == "" {
		reason = "No frontend-affecting files changed in this PR."
	}
	body := report.FormatPRCommentBody(report.ComparisonReport{}, report.PRCommentMeta{
		CommitSHA:     f.commitSHA,
		BaseSHA:       f.baseSHA,
		BaseBranch:    f.baseBranch,
		BuildURL:      f.buildURL,
		SkippedReason: reason,
	})
	if f.commentEnvelope {
		report.WriteCommentEnvelope(deps.stdout, body)
	} else {
		fmt.Fprint(deps.stdout, body)
	}
	return 0
}

func prepareCompareUpload(
	ctx context.Context,
	f cliFlags,
	deps cliDeps,
	headManifest capture.Manifest,
) (compareUploadPlan, error) {
	if f.gcsBucket == "" || (!f.uploadHeadSnapshot && !f.uploadReport) {
		return compareUploadPlan{}, nil
	}
	client, err := buildGCSClient(ctx, f, deps)
	if err != nil {
		return compareUploadPlan{}, err
	}

	headSHA := cmp.Or(f.commitSHA, headManifest.CommitSHA)
	plan := compareUploadPlan{
		client:             client,
		headSHA:            headSHA,
		fingerprint:        cmp.Or(f.fingerprint, headManifest.BuilderFingerprint),
		uploadHeadSnapshot: f.uploadHeadSnapshot,
		uploadReport:       f.uploadReport,
		inlineImages:       client.AccessMode() == gcs.AccessModePublic,
	}
	if f.uploadReport {
		runID := cmp.Or(f.runID, time.Now().UTC().Format("20060102-150405"))
		shaPart := cmp.Or(headSHA, "local")
		plan.reportPrefix = gcs.ReportPrefix(shaPart, runID)
		plan.reportBaseURL = client.ObjectURL(plan.reportPrefix)
	}
	return plan, nil
}

func executeCompareUpload(
	ctx context.Context,
	plan compareUploadPlan,
	headDir string,
	headManifest capture.Manifest,
	reportDir string,
) error {
	if plan.client == nil {
		return nil
	}
	if plan.uploadHeadSnapshot && plan.headSHA != "" && plan.fingerprint != "" && countCaptureErrors(headManifest) == 0 {
		if err := plan.client.UploadSnapshot(ctx, plan.fingerprint, plan.headSHA, headDir, headManifest); err != nil {
			return err
		}
	}
	if plan.uploadReport {
		if err := plan.client.UploadDirectory(ctx, plan.reportPrefix, reportDir); err != nil {
			return err
		}
	}
	return nil
}

func buildGCSClient(ctx context.Context, f cliFlags, deps cliDeps) (*gcs.Client, error) {
	if f.gcsBucket == "" {
		return nil, errors.New("--gcs-bucket is required for GCS operations")
	}
	mode, err := gcs.ParseAccessMode(f.gcsAccessMode)
	if err != nil {
		return nil, err
	}
	return deps.newGCSClientFn(ctx, f.gcsBucket, mode, f.workers)
}

func buildCaptureConfig(f cliFlags, outDir string, logWriter io.Writer) capture.Config {
	return capture.Config{
		StorybookDir:   f.storybookDir,
		OutputDir:      outDir,
		FilterPattern:  f.filter,
		Workers:        f.workers,
		ViewportWidth:  f.viewportWidth,
		ViewportHeight: f.viewportHeight,
		StoryTimeout:   f.storyTimeout,
		Retries:        f.retries,
		CommitSHA:      f.commitSHA,
		LogWriter:      logWriter,
	}
}

func countCaptureErrors(m capture.Manifest) int {
	cnt := 0
	for _, e := range m.Entries {
		if e.Error != "" {
			cnt++
		}
	}
	return cnt
}

func printCompareSummary(w io.Writer, rep report.ComparisonReport, reportDir string, reportBaseURL string) {
	fmt.Fprintf(
		w,
		"VRT Compare Summary: total=%d changed=%d added=%d removed=%d errored=%d unchanged=%d skipped=%d\n",
		rep.Summary.Total,
		rep.Summary.Changed,
		rep.Summary.Added,
		rep.Summary.Removed,
		rep.Summary.Errored,
		rep.Summary.Unchanged,
		rep.Summary.Skipped,
	)
	fmt.Fprintf(w, "Local HTML report: %s\n", filepath.Join(reportDir, report.ReportHTMLFileName))
	if reportBaseURL != "" {
		fmt.Fprintf(w, "GCS HTML report:   %s/%s\n", reportBaseURL, report.ReportHTMLFileName)
	}
}
