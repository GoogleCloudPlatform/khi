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

package qarunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

var (
	goReleaseTargets = []goBuildTarget{
		{goos: "windows", goarch: "amd64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64"},
		{goos: "darwin", goarch: "amd64"},
	}
	webTempBuildSpecs = []webTempBuildSpec{
		{
			source:   "build_web",
			tempPref: "khi-qa-build-web-*",
			summary:  "frontend production build failed",
			buildArgs: func(tmpDir string) []string {
				return []string{"ng", "build", "--output-path", tmpDir, "-c", "prod"}
			},
		},
		{
			source:   "build_web_dev",
			tempPref: "khi-qa-build-web-dev-*",
			summary:  "frontend dev build failed",
			buildArgs: func(tmpDir string) []string {
				return []string{"ng", "build", "--output-path", tmpDir, "--configuration", "dev"}
			},
		},
		{
			source:   "build_storybook",
			tempPref: "khi-qa-build-storybook-*",
			summary:  "frontend storybook build failed",
			buildArgs: func(tmpDir string) []string {
				return []string{"ng", "run", "frontend:build-storybook", "--output-dir", tmpDir}
			},
		},
	}
)

type goBuildTarget struct {
	goos   string
	goarch string
}

type webTempBuildSpec struct {
	source    string
	tempPref  string
	summary   string
	buildArgs func(tmpDir string) []string
}

func runSlowExternalChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) error {
	if scope.hasGo || scope.hasProto {
		if err := runGoSlowChecks(ctx, cfg, scope, rep); err != nil {
			return err
		}
	}
	if scope.hasWeb || scope.hasProto {
		if err := runWebSlowChecks(ctx, cfg, rep); err != nil {
			return err
		}
	}
	return nil
}

func runGoSlowChecks(ctx context.Context, cfg RunConfig, scope domainScope, rep *rule.QAReport) error {
	if cfg.UnusedExportsCheck != nil && !isSourceSkipped(cfg.SkipSources, "unused_exports", rep) {
		items, err := cfg.UnusedExportsCheck(cfg.RepoRoot, cfg.Paths)
		if err != nil {
			return err
		}
		rep.MustFix = append(rep.MustFix, items...)
	}
	if !isSourceSkipped(cfg.SkipSources, "deadcode", rep) {
		res, err := invokeCommand(ctx, cfg, commandSpec{
			dir:  cfg.RepoRoot,
			name: "go",
			args: []string{"tool", "deadcode", "-test", "./cmd/...", "./pkg/...", "./scripts/..."},
		})
		if err != nil {
			return err
		}
		appendDeadcodeFindings(rep, res, cfg.Paths)
	}
	if !isSourceSkipped(cfg.SkipSources, "go_test_race", rep) {
		args := append([]string{"test", "-race"}, scope.goPkgArgs()...)
		res, err := invokeCommand(ctx, cfg, commandSpec{
			dir:  cfg.RepoRoot,
			env:  []string{"KHI_SKIP_CLOUD_LOGGING=true"},
			name: "go",
			args: args,
		})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			appendCommandFailure(rep, "go_test_race", "go test -race failed", res)
		}
	}
	if !isSourceSkipped(cfg.SkipSources, "build_go", rep) {
		if err := runGoBuildCheck(ctx, cfg, rep); err != nil {
			return err
		}
	}
	return nil
}

func runGoBuildCheck(ctx context.Context, cfg RunConfig, rep *rule.QAReport) error {
	tmpDir, err := os.MkdirTemp("", "khi-qa-build-go-*")
	if err != nil {
		return fmt.Errorf("create temp dir for build_go: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outDir := tmpDir + string(os.PathSeparator)
	for _, target := range goReleaseTargets {
		res, err := invokeCommand(ctx, cfg, commandSpec{
			dir:  cfg.RepoRoot,
			env:  []string{"CGO_ENABLED=0", "GOOS=" + target.goos, "GOARCH=" + target.goarch},
			name: "go",
			args: []string{"build", "-o", outDir, "./cmd/kubernetes-history-inspector/..."},
		})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			appendCommandFailure(rep, "build_go", fmt.Sprintf("go build (%s/%s) failed", target.goos, target.goarch), res)
			return nil
		}
	}
	return nil
}

func runWebSlowChecks(ctx context.Context, cfg RunConfig, rep *rule.QAReport) error {
	webDir := filepath.Join(cfg.RepoRoot, "web")
	if !isSourceSkipped(cfg.SkipSources, "lint_unused", rep) {
		res, err := invokeCommand(ctx, cfg, commandSpec{dir: webDir, name: "npm", args: []string{"run", "lint:unused"}})
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			appendCommandFailure(rep, "lint_unused", "unused TypeScript exports detected", res)
		}
	}
	for _, spec := range webTempBuildSpecs {
		if isSourceSkipped(cfg.SkipSources, spec.source, rep) {
			continue
		}
		if err := runWebTempBuildCheck(ctx, cfg, spec, rep); err != nil {
			return err
		}
	}
	if !isSourceSkipped(cfg.SkipSources, "test_web", rep) {
		if err := runWebTestCheck(ctx, cfg, webDir, rep); err != nil {
			return err
		}
	}
	return nil
}

func runWebTempBuildCheck(ctx context.Context, cfg RunConfig, spec webTempBuildSpec, rep *rule.QAReport) error {
	tmpDir, err := os.MkdirTemp("", spec.tempPref)
	if err != nil {
		return fmt.Errorf("create temp dir for %s: %w", spec.source, err)
	}
	defer os.RemoveAll(tmpDir)

	res, err := invokeCommand(ctx, cfg, commandSpec{
		dir:  filepath.Join(cfg.RepoRoot, "web"),
		name: "npx",
		args: spec.buildArgs(tmpDir),
	})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		appendCommandFailure(rep, spec.source, spec.summary, res)
	}
	return nil
}

func runWebTestCheck(ctx context.Context, cfg RunConfig, webDir string, rep *rule.QAReport) error {
	var res CommandResult
	var err error
	if cfg.UseContainer {
		containerCmd, builderImage, resolveErr := gotestutil.ResolveContainerCommand()
		if resolveErr != nil {
			return resolveErr
		}
		res, err = invokeCommand(ctx, cfg, commandSpec{
			dir:  cfg.RepoRoot,
			name: containerCmd,
			args: []string{
				"run", "--rm", "--shm-size=1g",
				"-v", cfg.RepoRoot + ":/workspace",
				"-w", "/workspace/web",
				"-e", "HOME=/tmp",
				builderImage,
				"npx", "ng", "test", "--browsers", "ChromeHeadlessNoSandbox", "--watch=false", "--progress=false",
			},
		})
	} else {
		res, err = invokeCommand(ctx, cfg, commandSpec{
			dir:  webDir,
			name: "npx",
			args: []string{"ng", "test", "--browsers", "ChromeHeadlessNoSandbox", "--watch=false", "--progress=false"},
		})
	}
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		appendCommandFailure(rep, "test_web", "frontend unit tests failed", res)
	}
	return nil
}
