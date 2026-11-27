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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/covcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/mutcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/report"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

type dynamicStageInput struct {
	flags      cliFlags
	repoRoot   string
	modulePath string
	diffRes    gotestutil.DiffResult
	astRes     *astcheck.Result
	progress   *report.ProgressReporter
	errOut     io.Writer
}

func runDynamicStages(ctx context.Context, in dynamicStageInput, rep *rule.Report) error {
	workDir, err := os.MkdirTemp("", "khi-codequality-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary workspace: %w", err)
	}
	defer os.RemoveAll(workDir)

	covRes, err := runCoverageStage(ctx, in, workDir)
	if err != nil {
		return err
	}
	rep.Coverage = covRes.Summary
	rep.Findings = append(rep.Findings, covRes.Findings...)

	mutRes, err := runMutationStage(ctx, in, workDir, covRes)
	if err != nil {
		return err
	}
	rep.Mutation = mutRes.Summary
	rep.Findings = append(rep.Findings, mutRes.Findings...)
	rep.Suppressed = append(rep.Suppressed, mutRes.Suppressed...)
	return nil
}

func runCoverageStage(ctx context.Context, in dynamicStageInput, workDir string) (*covcheck.Result, error) {
	in.progress.Phase(2, 4, "Evaluating Go differential line coverage...")
	return covcheck.Evaluate(ctx, covcheck.Options{
		RepoRoot:               in.repoRoot,
		ModulePath:             in.modulePath,
		WorkDir:                filepath.Join(workDir, "cov"),
		Full:                   in.flags.full,
		PrecomputedProfilePath: in.flags.coverprofile,
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.astRes.DynamicChangedLines,
		ASTResult:              in.astRes,
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return os.ReadFile(filepath.Join(in.repoRoot, filepath.FromSlash(relPath)))
		},
		ErrOut: in.errOut,
	})
}

func runMutationStage(ctx context.Context, in dynamicStageInput, workDir string, covRes *covcheck.Result) (*mutcheck.Result, error) {
	return mutcheck.Evaluate(ctx, mutcheck.Options{
		RepoRoot:               in.repoRoot,
		WorkDir:                filepath.Join(workDir, "mut"),
		Thresholds:             rule.DefaultThresholds(),
		ChangedProdLinesByFile: in.astRes.DynamicChangedLines,
		ChangedTestLinesByFile: in.diffRes.ChangedTestLines,
		CoverBlocksByFile:      covRes.CoverBlocksByFile,
		HasCoverProfile:        covRes.HasCoverProfile,
		ASTResult:              in.astRes,
		Workers:                in.flags.workers,
		MaxMutants:             in.flags.maxMutants,
		Timeout:                in.flags.timeout,
		LogWriter:              in.errOut,
		Progress:               in.progress,
	})
}
