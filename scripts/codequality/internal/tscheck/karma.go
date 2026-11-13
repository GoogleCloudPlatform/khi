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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const (
	tsMinPerMutantTimeout  = 3 * time.Second
	tsconfigMutantTemplate = `{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "outDir": "./out-tsc/spec",
    "preserveSymlinks": true,
    "types": ["jasmine", "@types/gtag.js"]
  },
  "include": ["src/**/*.spec.ts", "src/**/*.d.ts"]
}
`
	angularJSONMutantTemplate = `{
  "version": 1,
  "newProjectRoot": "projects",
  "projects": {
    "frontend": {
      "projectType": "application",
      "root": "",
      "sourceRoot": "src",
      "prefix": "app",
      "architect": {
        "test": {
          "builder": "@angular/build:karma",
          "options": {
            "polyfills": ["zone.js", "zone.js/testing"],
            "tsConfig": "tsconfig.mutant.json",
            "karmaConfig": "karma.mutant.conf.js",
            "preserveSymlinks": true,
            "sourceMap": false,
            "inlineStyleLanguage": "sass",
            "assets": ["src/favicon.ico", "src/assets"],
            "styles": [
              "./node_modules/@angular/material/prebuilt-themes/indigo-pink.css",
              "src/styles.scss"
            ],
            "scripts": [%q]
          }
        }
      }
    }
  },
  "cli": {
    "analytics": false
  }
}
`
	karmaMutantConfTemplate = `module.exports = function (config) {
  config.set({
    basePath: '',
    frameworks: ['jasmine', 'khi-mutation'],
    plugins: [
      require('karma-jasmine'),
      require('karma-chrome-launcher'),
      require(%q),
    ],
    client: {
      jasmine: {
        random: false,
      },
      clearContext: true,
    },
    reporters: [],
    port: parseInt(process.env.KARMA_PORT, 10) || 9876,
    hostname: '127.0.0.1',
    listenAddress: '127.0.0.1',
    colors: false,
    logLevel: config.LOG_ERROR,
    autoWatch: false,
    singleRun: false,
    restartOnFileChange: false,
    captureTimeout: 180 * 1000,
    browserDisconnectTimeout: 30 * 1000,
    browserDisconnectTolerance: 0,
    browserNoActivityTimeout: 60 * 1000,
    customLaunchers: {
      ChromeHeadlessNoSandbox: {
        base: 'ChromeHeadless',
        flags: [
          '--no-sandbox',
          '--disable-extensions',
          '--enable-unsafe-swiftshader',
          '--enable-webgl',
          '--disable-gpu',
          '--remote-debugging-port=0',
        ],
      },
    },
  });
};
`
)

type tsStagedFile struct {
	plan    *TSFilePlan
	tsSites []TSMutationSite
}

type karmaWorkspaceRequest struct {
	repoRoot       string
	workDir        string
	stagedFiles    []tsStagedFile
	pendingMutants []*TSMutant
	maxMutantID    int
	maxProbeID     int
	maxMutants     int
	overallTimeout time.Duration
}

type karmaSessionRequest struct {
	repoRoot     string
	workDir      string
	cfgPath      string
	includeSpecs []string
	errOut       io.Writer
}

type karmaPluginConfig struct {
	CandidateIDs          []int             `json:"candidateIds"`
	FileByMutantID        map[string]string `json:"fileByMutantId"`
	MaxMutants            int               `json:"maxMutants"`
	MinPerMutantTimeoutMs int               `json:"minPerMutantTimeoutMs"`
	OverallTimeoutMs      int               `json:"overallTimeoutMs"`
	ResultsPath           string            `json:"resultsPath"`
}

type karmaPluginResults struct {
	BaselineFailed bool                                  `json:"baselineFailed"`
	ErrorMessage   string                                `json:"errorMessage"`
	Outcomes       map[string]string                     `json:"outcomes"`
	SpecOrder      []redundancy.TSSpecOrderEntry         `json:"specOrder"`
	SpecPolarities map[string][]string                   `json:"specPolarities"`
	MutantRuns     map[string]redundancy.TSMutantSpecRun `json:"mutantRuns"`
}

type containerMutCmdSpec struct {
	absRepo   string
	absWork   string
	tmpWebDir string
	port      string
	cfgPath   string
	ngArgs    []string
}

func prepareTSMutationWorkspace(req karmaWorkspaceRequest) (string, string, error) {
	absRepo, err := filepath.Abs(req.repoRoot)
	if err != nil {
		return "", "", err
	}
	absWork, err := filepath.Abs(req.workDir)
	if err != nil {
		return "", "", err
	}
	tmpWebDir := filepath.Join(absWork, "web")
	if err := stageWebSymlinkMirror(absRepo, tmpWebDir); err != nil {
		return "", "", err
	}
	if err := writeMutatedWebFiles(tmpWebDir, req.stagedFiles, req.maxMutantID, req.maxProbeID); err != nil {
		return "", "", err
	}
	if err := writeKarmaHarnessConfigs(absRepo, tmpWebDir); err != nil {
		return "", "", err
	}
	return writeKarmaPluginConfigFile(absWork, req)
}

func stageWebSymlinkMirror(absRepo string, tmpWebDir string) error {
	if err := os.MkdirAll(tmpWebDir, 0o755); err != nil {
		return err
	}
	for _, entry := range []string{"node_modules", "package.json", "tsconfig.json"} {
		srcPath := filepath.Join(absRepo, "web", entry)
		dstPath := filepath.Join(tmpWebDir, entry)
		if err := os.Symlink(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to symlink %s: %w", entry, err)
		}
	}
	if err := createSymlinkTree(filepath.Join(absRepo, "web", "src"), filepath.Join(tmpWebDir, "src")); err != nil {
		return fmt.Errorf("failed to stage symlink tree for web/src: %w", err)
	}
	return nil
}

func writeMutatedWebFiles(tmpWebDir string, staged []tsStagedFile, maxMutantID int, maxProbeID int) error {
	for _, tp := range staged {
		relUnderWeb := strings.TrimPrefix(tp.plan.File, "web/")
		dstFile := filepath.Join(tmpWebDir, filepath.FromSlash(relUnderWeb))
		if err := os.Remove(dstFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove symlink before writing mutant %s: %w", dstFile, err)
		}
		rendered := RenderTSSuperMutantSource(tp.plan.Source, tp.tsSites)
		if err := os.WriteFile(dstFile, rendered, 0o600); err != nil {
			return fmt.Errorf("failed to write Super-Mutant TypeScript file %s: %w", dstFile, err)
		}
	}
	runtimePath := filepath.Join(tmpWebDir, tsBrowserRuntimeFileName)
	return os.WriteFile(runtimePath, GenerateTSBrowserRuntimeSource(maxMutantID, maxProbeID), 0o600)
}

func writeKarmaHarnessConfigs(absRepo string, tmpWebDir string) error {
	if err := os.WriteFile(filepath.Join(tmpWebDir, "tsconfig.mutant.json"), []byte(tsconfigMutantTemplate), 0o600); err != nil {
		return err
	}
	angularJSON := fmt.Sprintf(angularJSONMutantTemplate, tsBrowserRuntimeFileName)
	if err := os.WriteFile(filepath.Join(tmpWebDir, "angular.json"), []byte(angularJSON), 0o600); err != nil {
		return err
	}
	pluginScriptPath, err := resolveTSHelperScriptPath(absRepo, "karma_mutation_plugin.js")
	if err != nil {
		return err
	}
	karmaConf := fmt.Sprintf(karmaMutantConfTemplate, pluginScriptPath)
	return os.WriteFile(filepath.Join(tmpWebDir, "karma.mutant.conf.js"), []byte(karmaConf), 0o600)
}

func writeKarmaPluginConfigFile(absWork string, req karmaWorkspaceRequest) (string, string, error) {
	resultsPath := filepath.Join(absWork, "mutation-results.json")
	cfgPath := filepath.Join(absWork, "mutation-config.json")
	candIDs := make([]int, 0, len(req.pendingMutants))
	fileByID := make(map[string]string, len(req.pendingMutants))
	for _, m := range req.pendingMutants {
		candIDs = append(candIDs, m.ID)
		fileByID[strconv.Itoa(m.ID)] = m.File
	}
	pluginBudget := max(15*time.Second, req.overallTimeout-30*time.Second)
	pluginCfg := karmaPluginConfig{
		CandidateIDs:          candIDs,
		FileByMutantID:        fileByID,
		MaxMutants:            req.maxMutants,
		MinPerMutantTimeoutMs: int(tsMinPerMutantTimeout.Milliseconds()),
		OverallTimeoutMs:      int(pluginBudget.Milliseconds()),
		ResultsPath:           resultsPath,
	}
	cfgBytes, err := json.MarshalIndent(pluginCfg, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return "", "", err
	}
	return resultsPath, cfgPath, nil
}

func runKarmaMutationSession(ctx context.Context, req karmaSessionRequest) error {
	port, err := gotestutil.PickKarmaPort(ctx)
	if err != nil {
		return err
	}
	absRepo, err := filepath.Abs(req.repoRoot)
	if err != nil {
		return err
	}
	absWork, err := filepath.Abs(req.workDir)
	if err != nil {
		return err
	}
	tmpWebDir := filepath.Join(absWork, "web")
	ngArgs := buildKarmaMutationArgs(req.includeSpecs)

	var cmd *exec.Cmd
	if gotestutil.IsContainerExecutionEnabled() {
		cmd, err = buildContainerKarmaMutationCmd(ctx, containerMutCmdSpec{
			absRepo:   absRepo,
			absWork:   absWork,
			tmpWebDir: tmpWebDir,
			port:      port,
			cfgPath:   req.cfgPath,
			ngArgs:    ngArgs,
		})
		if err != nil {
			return err
		}
	} else {
		cmd = exec.CommandContext(ctx, "npx", ngArgs...)
		cmd.Dir = tmpWebDir
		cmd.Env = append(os.Environ(), "KARMA_PORT="+port, "KHI_MUTATION_CONFIG_FILE="+req.cfgPath)
	}

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		if req.errOut != nil {
			_, _ = io.Copy(req.errOut, &combined)
		}
		return fmt.Errorf("TypeScript Karma mutation run failed (npx %s): %w", strings.Join(ngArgs, " "), err)
	}
	return nil
}

func buildKarmaMutationArgs(includeSpecs []string) []string {
	ngArgs := make([]string, 0, 5+len(includeSpecs))
	ngArgs = append(ngArgs,
		"ng", "test",
		"--browsers=ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
	)
	for _, spec := range includeSpecs {
		ngArgs = append(ngArgs, "--include="+spec)
	}
	return ngArgs
}

func buildContainerKarmaMutationCmd(ctx context.Context, spec containerMutCmdSpec) (*exec.Cmd, error) {
	containerCmd, builderImage, err := gotestutil.ResolveContainerCommand()
	if err != nil {
		return nil, err
	}
	args := []string{
		"run", "--rm",
		"--shm-size=1g",
		fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
		fmt.Sprintf("-v=%s:%s:ro", spec.absRepo, spec.absRepo),
		fmt.Sprintf("-v=%s:%s", spec.absWork, spec.absWork),
		"-w=" + spec.tmpWebDir,
		"-e=HOME=/tmp",
		"-e=KARMA_PORT=" + spec.port,
		"-e=KHI_MUTATION_CONFIG_FILE=" + spec.cfgPath,
		builderImage,
		"npx",
	}
	args = append(args, spec.ngArgs...)
	cmd := exec.CommandContext(ctx, containerCmd, args...)
	cmd.Dir = spec.absRepo
	return cmd, nil
}

func createSymlinkTree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(absPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcRoot, absPath)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return os.Symlink(absPath, target)
	})
}

func applyKarmaOutcomes(pending []*TSMutant, outcomes map[string]string) {
	for _, m := range pending {
		switch outcomes[strconv.Itoa(m.ID)] {
		case string(rule.MutantKilled):
			m.Status = rule.MutantKilled
		case string(rule.MutantTimedOut):
			m.Status = rule.MutantTimedOut
		case string(rule.MutantSurvived):
			m.Status = rule.MutantSurvived
		case string(rule.MutantUncovered):
			m.Status = rule.MutantUncovered
		default:
			m.Status = rule.MutantSkippedBudget
		}
	}
}
