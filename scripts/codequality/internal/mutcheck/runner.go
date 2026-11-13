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

package mutcheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

const (
	minPerMutantTimeout     = 3 * time.Second
	baselineTimeoutFactor   = 5
	fastVerboseBaselineCap  = 500 * time.Millisecond
	maxTracedSubtestsPerPkg = 40
)

// ProgressHooks reports compilation, subtest tracing, and mutant execution progress.
type ProgressHooks interface {
	// Phase updates the current pipeline phase label and step progress.
	Phase(step int, totalSteps int, message string)
	// SubtestProgress reports progress while tracing leaf subtest polarities.
	SubtestProgress(pkgLabel string, done int, total int)
	// StartMutantPool announces the start of parallel mutant execution.
	StartMutantPool(total int, workers int)
	// RecordMutantResult records one completed mutant evaluation status.
	RecordMutantResult(status rule.MutantStatus)
}

type flatSiteRef struct {
	fileIdx int
	siteIdx int
}

type compileRequest struct {
	repoRoot    string
	pkgDir      string
	pkgPlans    []*FilePlan
	maxMutantID int
	pkgWorkDir  string
	logWriter   io.Writer
}

type baselineRequest struct {
	binPath   string
	pkgAbsDir string
	tracePath string
	verbose   bool
}

func compilePackageWithFallback(ctx context.Context, req compileRequest) (string, error) {
	allRefs := collectFlatSiteRefs(req.pkgPlans)
	if len(allRefs) == 0 {
		return "", nil
	}
	binPath := filepath.Join(req.pkgWorkDir, "pkg.test")
	if err := tryCompileSubset(ctx, req, allRefs, filepath.Join(req.pkgWorkDir, "full"), binPath); err == nil {
		return binPath, nil
	}
	if err := verifyCleanPackageCompiles(ctx, req); err != nil {
		return "", err
	}
	if req.logWriter != nil {
		fmt.Fprintf(req.logWriter, "Package ./%s had Super-Mutant compile edge case; bisecting sites...\n", req.pkgDir)
	}
	attempts := 0
	validRefs := bisectCompilableRefs(ctx, req, allRefs, &attempts)
	markDroppedSitesCompileSkipped(req.pkgPlans, validRefs)
	if len(validRefs) == 0 {
		return "", nil
	}
	if err := tryCompileSubset(ctx, req, validRefs, filepath.Join(req.pkgWorkDir, "final"), binPath); err != nil {
		return "", fmt.Errorf("failed to compile isolated mutant subset for ./%s: %w", req.pkgDir, err)
	}
	return binPath, nil
}

func collectFlatSiteRefs(plans []*FilePlan) []flatSiteRef {
	var refs []flatSiteRef
	for fIdx, p := range plans {
		for sIdx := range p.Sites {
			refs = append(refs, flatSiteRef{fileIdx: fIdx, siteIdx: sIdx})
		}
	}
	return refs
}

func verifyCleanPackageCompiles(ctx context.Context, req compileRequest) error {
	cleanBin := filepath.Join(req.pkgWorkDir, "clean.test")
	if err := runGoTestCompile(ctx, req.repoRoot, "", cleanBin, req.pkgDir); err != nil {
		return fmt.Errorf("package ./%s failed to compile before mutation: %w", req.pkgDir, err)
	}
	return nil
}

func bisectCompilableRefs(ctx context.Context, req compileRequest, refs []flatSiteRef, attempts *int) []flatSiteRef {
	if len(refs) == 0 || ctx.Err() != nil {
		return nil
	}
	*attempts++
	dir := filepath.Join(req.pkgWorkDir, fmt.Sprintf("bisect_%d", *attempts))
	if err := tryCompileSubset(ctx, req, refs, dir, filepath.Join(dir, "sub.test")); err == nil {
		return refs
	}
	if len(refs) == 1 {
		return nil
	}
	mid := len(refs) / 2
	left := bisectCompilableRefs(ctx, req, refs[:mid], attempts)
	right := bisectCompilableRefs(ctx, req, refs[mid:], attempts)
	return append(left, right...)
}

func markDroppedSitesCompileSkipped(plans []*FilePlan, validRefs []flatSiteRef) {
	validSet := make(map[flatSiteRef]struct{}, len(validRefs))
	for _, r := range validRefs {
		validSet[r] = struct{}{}
	}
	for fIdx, p := range plans {
		for sIdx := range p.Sites {
			if _, ok := validSet[flatSiteRef{fileIdx: fIdx, siteIdx: sIdx}]; !ok {
				for _, m := range p.Sites[sIdx].Mutants {
					m.Status = rule.MutantCompileSkipped
				}
			}
		}
	}
}

func tryCompileSubset(ctx context.Context, req compileRequest, refs []flatSiteRef, workDir string, outBin string) error {
	sitesPerFile := make([][]MutationSite, len(req.pkgPlans))
	for _, r := range refs {
		sitesPerFile[r.fileIdx] = append(sitesPerFile[r.fileIdx], req.pkgPlans[r.fileIdx].Sites[r.siteIdx])
	}
	specs := make([]OverlayFileSpec, 0, len(req.pkgPlans))
	for i, p := range req.pkgPlans {
		if len(sitesPerFile[i]) > 0 {
			specs = append(specs, OverlayFileSpec{RelFile: p.File, Source: p.Source, Sites: sitesPerFile[i]})
		}
	}
	overlayPath, err := WritePackageOverlay(OverlayRequest{
		RepoRoot:    req.repoRoot,
		PkgRelDir:   req.pkgDir,
		PkgName:     req.pkgPlans[0].PackageName,
		Files:       specs,
		MaxMutantID: req.maxMutantID,
		WorkDir:     workDir,
	})
	if err != nil {
		return err
	}
	return runGoTestCompile(ctx, req.repoRoot, overlayPath, outBin, req.pkgDir)
}

func runGoTestCompile(ctx context.Context, repoRoot string, overlayPath string, outBin string, pkgDir string) error {
	pkgTarget := "./" + strings.TrimPrefix(pkgDir, "./")
	if pkgDir == "." || pkgDir == "" {
		pkgTarget = "."
	}
	args := []string{"test"}
	if overlayPath != "" {
		args = append(args, "-overlay="+overlayPath)
	}
	args = append(args, "-c", "-o", outBin, pkgTarget)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = buildMutantEnv(os.Environ(), 0, "")
	var stderr bytes.Buffer
	cmd.Stdout = &stderr
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runBaselineTestBinary(ctx context.Context, req baselineRequest) (time.Duration, []byte, error) {
	start := time.Now()
	var args []string
	if req.verbose {
		args = append(args, "-test.v")
	}
	cmd := exec.CommandContext(ctx, req.binPath, args...)
	cmd.Dir = req.pkgAbsDir
	cmd.Env = buildMutantEnv(os.Environ(), 0, req.tracePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return 0, nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return time.Since(start), out.Bytes(), nil
}

func groupFilesByPackage(changed map[string]map[int]struct{}) (map[string][]string, []string) {
	byPkg := make(map[string][]string)
	var pkgs []string
	for relFile, lines := range changed {
		if len(lines) == 0 {
			continue
		}
		clean := strings.TrimPrefix(filepath.ToSlash(relFile), "./")
		pkgDir := path.Dir(clean)
		if _, ok := byPkg[pkgDir]; !ok {
			pkgs = append(pkgs, pkgDir)
		}
		byPkg[pkgDir] = append(byPkg[pkgDir], clean)
	}
	slices.Sort(pkgs)
	for _, pkg := range pkgs {
		slices.Sort(byPkg[pkg])
	}
	return byPkg, pkgs
}

func computePerMutantTimeout(baseline time.Duration) time.Duration {
	return max(baseline*baselineTimeoutFactor, minPerMutantTimeout)
}

func resolveWorkerCount(requested int) int {
	if requested > 0 {
		return requested
	}
	return max(runtime.NumCPU(), 1)
}

func buildMutantEnv(base []string, mutantID int, tracePath string) []string {
	env := make([]string, 0, len(base)+3)
	hasSkipCloudLogging := false
	for _, kv := range base {
		switch {
		case strings.HasPrefix(kv, "KHI_MUTANT_ID="), strings.HasPrefix(kv, "KHI_MUTANT_TRACE_FILE="):
			continue
		case strings.HasPrefix(kv, "KHI_SKIP_CLOUD_LOGGING="):
			hasSkipCloudLogging = true
		}
		env = append(env, kv)
	}
	if !hasSkipCloudLogging {
		env = append(env, "KHI_SKIP_CLOUD_LOGGING=true")
	}
	env = append(env, "KHI_MUTANT_ID="+strconv.Itoa(mutantID))
	if tracePath != "" {
		env = append(env, "KHI_MUTANT_TRACE_FILE="+tracePath)
	}
	return env
}
