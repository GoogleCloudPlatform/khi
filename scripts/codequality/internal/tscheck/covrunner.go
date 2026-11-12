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

package tscheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// KarmaCoverageRequest configures an Angular Karma coverage execution.
type KarmaCoverageRequest struct {
	RepoRoot    string
	Plan        gotestutil.TSTestPlan
	CoverageDir string
	ErrOut      io.Writer
}

func loadOrRunTSCoverage(ctx context.Context, opts Options) (map[string]gotestutil.IstanbulFileCoverage, bool, error) {
	changedFiles := sortedTSProdFiles(opts.ChangedLinesByFile)
	covByFile := make(map[string]gotestutil.IstanbulFileCoverage)
	hasTSCov := false

	if opts.TSCoveragePath != "" {
		if info, err := os.Stat(opts.TSCoveragePath); err == nil && !info.IsDir() {
			phaseIfPresent(opts.Progress, 2, 4, fmt.Sprintf("Loading Istanbul coverage %s...", filepath.Base(opts.TSCoveragePath)))
			loaded, loadErr := gotestutil.LoadTSCoverageIfPresent(opts.TSCoveragePath, opts.RepoRoot)
			if loadErr != nil {
				return nil, false, loadErr
			}
			covByFile = loaded
			hasTSCov = true
		}
	} else if len(changedFiles) > 0 || opts.Full {
		var err error
		covByFile, hasTSCov, err = executeFreshKarmaCoverage(ctx, opts, changedFiles)
		if err != nil {
			return nil, false, err
		}
	}

	if hasTSCov {
		if err := SupplementMissingTSCoverage(ctx, opts.RepoRoot, changedFiles, covByFile); err != nil {
			return nil, false, err
		}
	}
	return covByFile, hasTSCov, nil
}

func executeFreshKarmaCoverage(
	ctx context.Context,
	opts Options,
	changedFiles []string,
) (map[string]gotestutil.IstanbulFileCoverage, bool, error) {
	plan, err := gotestutil.PlanTSTestExecution(opts.RepoRoot, changedFiles, opts.Full)
	if err != nil {
		return nil, false, err
	}
	if plan.SkipKarma {
		return make(map[string]gotestutil.IstanbulFileCoverage), true, nil
	}
	phaseIfPresent(opts.Progress, 2, 4, fmt.Sprintf("Running TypeScript coverage across %d spec(s)...", len(plan.IncludeSpecs)))
	covDir, err := os.MkdirTemp("", "khi-ts-cov-*")
	if err != nil {
		return nil, false, err
	}
	defer os.RemoveAll(covDir)

	runCov := opts.RunKarmaCoverage
	if runCov == nil {
		runCov = RunTSTestCoverage
	}
	if err := runCov(ctx, KarmaCoverageRequest{
		RepoRoot:    opts.RepoRoot,
		Plan:        plan,
		CoverageDir: covDir,
		ErrOut:      opts.LogWriter,
	}); err != nil {
		return nil, false, err
	}
	loaded, err := gotestutil.LoadTSCoverageIfPresent(filepath.Join(covDir, "coverage-final.json"), opts.RepoRoot)
	if err != nil {
		return nil, false, err
	}
	return loaded, true, nil
}

// RunTSTestCoverage executes Angular Karma tests with coverage output written to req.CoverageDir.
func RunTSTestCoverage(ctx context.Context, req KarmaCoverageRequest) error {
	if req.Plan.SkipKarma {
		return nil
	}
	port, err := gotestutil.PickKarmaPort(ctx)
	if err != nil {
		return err
	}
	ngArgs := buildKarmaCoverageArgs(req.Plan.IncludeSpecs)

	var cmd *exec.Cmd
	if gotestutil.IsContainerExecutionEnabled() {
		cmd, err = buildContainerKarmaCoverageCommand(ctx, req.RepoRoot, req.CoverageDir, port, ngArgs)
		if err != nil {
			return err
		}
	} else {
		cmd = exec.CommandContext(ctx, "npx", ngArgs...)
		cmd.Dir = filepath.Join(req.RepoRoot, "web")
		cmd.Env = append(os.Environ(), "KARMA_PORT="+port, "KARMA_COVERAGE_DIR="+req.CoverageDir)
	}

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		if req.ErrOut != nil {
			_, _ = io.Copy(req.ErrOut, &combined)
		}
		return fmt.Errorf("ng test failed (npx %s): %w", strings.Join(ngArgs, " "), err)
	}
	return nil
}

func buildKarmaCoverageArgs(includeSpecs []string) []string {
	ngArgs := make([]string, 0, 6+len(includeSpecs))
	ngArgs = append(ngArgs,
		"ng", "test",
		"--code-coverage",
		"--browsers", "ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
	)
	for _, spec := range includeSpecs {
		ngArgs = append(ngArgs, "--include="+spec)
	}
	return ngArgs
}

func buildContainerKarmaCoverageCommand(
	ctx context.Context,
	repoRoot string,
	coverageDir string,
	port string,
	ngArgs []string,
) (*exec.Cmd, error) {
	containerCmd, builderImage, err := gotestutil.ResolveContainerCommand()
	if err != nil {
		return nil, err
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	absCov, err := filepath.Abs(coverageDir)
	if err != nil {
		return nil, err
	}
	args := []string{
		"run", "--rm",
		"--shm-size=1g",
		fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
		fmt.Sprintf("-v=%s:/workspace", absRepo),
		fmt.Sprintf("-v=%s:%s", absCov, absCov),
		"-w=/workspace/web",
		"-e=HOME=/tmp",
		"-e=KARMA_PORT=" + port,
		"-e=KARMA_COVERAGE_DIR=" + absCov,
		builderImage,
		"npx",
	}
	args = append(args, ngArgs...)
	cmd := exec.CommandContext(ctx, containerCmd, args...)
	cmd.Dir = absRepo
	return cmd, nil
}

// SupplementMissingTSCoverage populates zero-hit Istanbul coverage entries for unimported changed TypeScript files.
func SupplementMissingTSCoverage(
	ctx context.Context,
	repoRoot string,
	changedTSFiles []string,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
) error {
	missing := collectUnimportedTSFiles(repoRoot, changedTSFiles, covByFile)
	if len(missing) == 0 {
		return nil
	}
	scriptPath, err := resolveTSHelperScriptPath(repoRoot, "zero_hit_ts.js")
	if err != nil {
		return err
	}
	args := make([]string, 0, 2+len(missing))
	args = append(args, scriptPath, repoRoot)
	args = append(args, missing...)

	outBytes, err := gotestutil.RunCommandOutput(ctx, repoRoot, "node", args...)
	if err != nil {
		return fmt.Errorf("failed to generate zero-hit TypeScript coverage: %w", err)
	}
	supplemented, err := gotestutil.ParseIstanbulCoverageJSON(bytes.NewReader(outBytes), repoRoot)
	if err != nil {
		return err
	}
	for file, fc := range supplemented {
		covByFile[file] = fc
	}
	return nil
}

func collectUnimportedTSFiles(
	repoRoot string,
	changedTSFiles []string,
	covByFile map[string]gotestutil.IstanbulFileCoverage,
) []string {
	var missing []string
	for _, rawFile := range changedTSFiles {
		clean := strings.TrimPrefix(filepath.ToSlash(rawFile), "./")
		if !gotestutil.IsTargetProductionTSFile(clean) {
			continue
		}
		if _, exists := covByFile[clean]; exists {
			continue
		}
		absFile := filepath.Join(repoRoot, filepath.FromSlash(clean))
		if info, err := os.Stat(absFile); err == nil && !info.IsDir() {
			missing = append(missing, clean)
		}
	}
	return missing
}

func resolveTSHelperScriptPath(repoRoot string, scriptName string) (string, error) {
	repoCandidate := filepath.Join(repoRoot, "scripts", "codequality", "internal", "tscheck", "js", scriptName)
	if _, err := os.Stat(repoCandidate); err == nil {
		return repoCandidate, nil
	}
	_, callerFile, _, _ := runtime.Caller(0)
	callerCandidate := filepath.Join(filepath.Dir(callerFile), "js", scriptName)
	if _, err := os.Stat(callerCandidate); err == nil {
		return callerCandidate, nil
	}
	return "", fmt.Errorf("%s helper script not found under %s", scriptName, repoRoot)
}
