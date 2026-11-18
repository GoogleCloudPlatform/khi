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
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/gcs"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/imagediff"
)

const (
	modeFingerprint      = "fingerprint"
	modeHasSnapshot      = "has-snapshot"
	modeDownloadSnapshot = "download-snapshot"
	modeCapture          = "capture"
	modeCompare          = "compare"
	modeSkipComment      = "skip-comment"

	exitCodeCacheMiss = 2
)

type cliFlags struct {
	mode               string
	storybookDir       string
	outDir             string
	baseDir            string
	headDir            string
	reportDir          string
	filter             string
	workers            int
	viewportWidth      int
	viewportHeight     int
	storyTimeout       time.Duration
	retries            int
	colorThreshold     float64
	maxDiffPixelRatio  float64
	includeAA          bool
	gcsBucket          string
	gcsAccessMode      string
	fingerprint        string
	commitSHA          string
	baseSHA            string
	baseBranch         string
	runID              string
	buildURL           string
	uploadHeadSnapshot bool
	uploadReport       bool
	baselineCacheHit   bool
	commentEnvelope    bool
	skipReason         string
	failOnDiff         bool
}

func parseFlags(args []string, stderr io.Writer) (cliFlags, error) {
	fs := flag.NewFlagSet("vrt", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var f cliFlags
	fs.StringVar(&f.mode, "mode", modeCapture, "execution mode: fingerprint, has-snapshot, download-snapshot, capture, compare, skip-comment")
	fs.StringVar(&f.storybookDir, "storybook-dir", "web/storybook-static", "path to built storybook-static directory")
	fs.StringVar(&f.outDir, "out-dir", "vrt-output/snapshot", "output directory for captured or downloaded snapshots")
	fs.StringVar(&f.baseDir, "base-dir", "", "local baseline snapshot directory for compare mode")
	fs.StringVar(&f.headDir, "head-dir", "", "local candidate snapshot directory for compare mode")
	fs.StringVar(&f.reportDir, "report-dir", "vrt-output/report", "output directory for visual diff report artifacts")
	fs.StringVar(&f.filter, "filter", "", "optional case-insensitive regex matching story ID, title, or importPath")
	fs.IntVar(&f.workers, "workers", 0, "parallel browser/diff worker count (0 = auto)")
	fs.IntVar(&f.viewportWidth, "viewport-width", capture.DefaultViewportWidth, "default viewport width in pixels")
	fs.IntVar(&f.viewportHeight, "viewport-height", capture.DefaultViewportHeight, "default viewport height in pixels")
	fs.DurationVar(&f.storyTimeout, "story-timeout", capture.DefaultStoryTimeout, "timeout per story capture")
	fs.IntVar(&f.retries, "retries", capture.DefaultRetries, "retry attempts on transient story capture error")
	fs.Float64Var(&f.colorThreshold, "color-threshold", imagediff.DefaultColorThreshold, "YIQ perceptual diff color threshold in [0,1]")
	fs.Float64Var(&f.maxDiffPixelRatio, "max-diff-ratio", imagediff.DefaultMaxDiffPixelRatio, "maximum differing pixel ratio allowed as unchanged")
	fs.BoolVar(&f.includeAA, "include-aa", false, "treat anti-aliased edge pixels as visual diffs")
	fs.StringVar(&f.gcsBucket, "gcs-bucket", "", "GCS bucket name for VRT snapshot cache and reports")
	fs.StringVar(&f.gcsAccessMode, "gcs-access-mode", string(gcs.AccessModeConsole), "GCS link format: console (internal GCP) or public (storage.googleapis.com)")
	fs.StringVar(&f.fingerprint, "fingerprint", "", "builder fingerprint cache key segment")
	fs.StringVar(&f.commitSHA, "commit-sha", "", "Git commit SHA for snapshot/report metadata")
	fs.StringVar(&f.baseSHA, "base-sha", "", "baseline Git merge-base commit SHA")
	fs.StringVar(&f.baseBranch, "base-branch", "", "base branch name for PR comment footer")
	fs.StringVar(&f.runID, "run-id", "", "unique CI build/run identifier for report GCS path")
	fs.StringVar(&f.buildURL, "build-url", "", "Cloud Build execution log URL for PR comment footer")
	fs.BoolVar(&f.uploadHeadSnapshot, "upload-head-snapshot", false, "upload captured HEAD snapshot to GCS cache")
	fs.BoolVar(&f.uploadReport, "upload-report", false, "upload generated comparison report directory to GCS")
	fs.BoolVar(&f.baselineCacheHit, "baseline-cache-hit", false, "record whether baseline was loaded from GCS cache")
	fs.BoolVar(&f.commentEnvelope, "comment-envelope", false, "emit Base64 sticky PR comment envelope to stdout")
	fs.StringVar(&f.skipReason, "skip-reason", "", "human-readable reason emitted in skip-comment mode")
	fs.BoolVar(&f.failOnDiff, "fail-on-diff", false, "exit non-zero when visual diffs are detected")

	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	f.mode = strings.ToLower(strings.TrimSpace(f.mode))
	if err := validateFlags(f); err != nil {
		return cliFlags{}, err
	}
	return f, nil
}

func validateFlags(f cliFlags) error {
	switch f.mode {
	case modeFingerprint, modeHasSnapshot, modeDownloadSnapshot, modeCapture, modeCompare, modeSkipComment:
	default:
		return fmt.Errorf("unsupported mode %q", f.mode)
	}
	if _, err := gcs.ParseAccessMode(f.gcsAccessMode); err != nil {
		return err
	}
	return nil
}
