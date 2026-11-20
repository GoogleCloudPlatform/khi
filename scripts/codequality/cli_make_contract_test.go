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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	makeContinuationPattern  = regexp.MustCompile(`\\\r?\n[ \t]*`)
	makeCommentPattern       = regexp.MustCompile(`(?m)#.*$`)
	makeLangAssignPattern    = regexp.MustCompile(`(?m)^\s*LANG\s*(?:[:+?]?=)`)
	makeLangRefPattern       = regexp.MustCompile(`\$\(\s*LANG\s*\)|\$\{\s*LANG\s*\}`)
	shellContinuationPattern = regexp.MustCompile(`\\\r?\n[ \t]*`)
	whitespaceRunPattern     = regexp.MustCompile(`\s+`)
	jsTokenPattern           = regexp.MustCompile(`(?s)"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|` + "`(?:\\\\.|[^`\\\\])*`" + `|/\*.*?\*/|//[^\r\n]*`)
	jsProjectKeyPattern      = regexp.MustCompile(`(?m)\bproject\s*:`)
	commentRunPattern        = regexp.MustCompile(`khi-codequality"?\s+check\b`)
	prePushQATargetPattern   = regexp.MustCompile(`\bqa-all(?:-must)?\b`)
)

type contractFileRule struct {
	name            string
	relPath         string
	mustContainNorm []string
	mustNotContain  []string
	mustMatchNorm   []*regexp.Regexp
	customCheck     func(t *testing.T, content string)
}

// TestContractMakefilesAndScripts verifies Make target reorganization, CI/workflow command wiring, and hook script updates.
func TestContractMakefilesAndScripts(t *testing.T) {
	repoRoot := findContractRepoRoot(t)
	rules := []contractFileRule{
		// Verifies scripts/make/qa.mk defines all 9 phony qa targets, QA_LANG, opt-in flags, codequality.mk include, and zero qa-fast prerequisites.
		{
			name:    "qa.mk defines nine phony targets and isolated fast prerequisites",
			relPath: filepath.Join("scripts", "make", "qa.mk"),
			mustContainNorm: []string{
				"QA_LANG",
				".config/khi/codequality.mk",
				"CODEQUALITY_SIMILARITY",
				"CODEQUALITY_DECISIONS",
				"go run ./scripts/codequality qa",
			},
			customCheck: verifyQAMakefileContract,
		},
		// Verifies scripts/make/testing.mk drops legacy codequality targets while retaining CI targets.
		{
			name:        "testing.mk removes legacy duplicate targets and keeps CI targets",
			relPath:     filepath.Join("scripts", "make", "testing.mk"),
			customCheck: verifyTestingMakefileContract,
		},
		// Verifies scripts/make/lint.mk invokes conventions and unused-exports subcommands without legacy flags.
		{
			name:    "lint.mk uses conventions and unused-exports subcommands",
			relPath: filepath.Join("scripts", "make", "lint.mk"),
			mustContainNorm: []string{
				"scripts/codequality conventions",
				"scripts/codequality unused-exports",
			},
			mustNotContain: []string{"--conventions", "--unused-exports"},
		},
		// Verifies pullrequest.yaml invokes conventions and unused-exports subcommands without legacy flags.
		{
			name:    "pullrequest.yaml uses conventions and unused-exports subcommands",
			relPath: filepath.Join(".github", "workflows", "pullrequest.yaml"),
			mustContainNorm: []string{
				"scripts/codequality conventions",
				"scripts/codequality unused-exports",
			},
			mustNotContain: []string{"--conventions", "--unused-exports"},
		},
		// Verifies codequality-comment.yaml invokes check --speed=fast --filter=advisory --decisions without --decisions-only.
		{
			name:    "codequality-comment.yaml uses check fast advisory with decisions opt-in",
			relPath: filepath.Join(".github", "workflows", "codequality-comment.yaml"),
			mustContainNorm: []string{
				"--speed=fast",
				"--filter=advisory",
				"--decisions",
			},
			mustMatchNorm:  []*regexp.Regexp{commentRunPattern},
			mustNotContain: []string{"--decisions-only"},
		},
		// Verifies run-codequality-ci.sh invokes check --speed=all --filter=all with opt-in --similarity and without --skip-sources workaround.
		{
			name:    "run-codequality-ci.sh uses check all all with opt-in similarity",
			relPath: filepath.Join("scripts", "run-codequality-ci.sh"),
			mustContainNorm: []string{
				"scripts/codequality check",
				"--speed=all",
				"--filter=all",
				"--similarity",
			},
			mustNotContain: []string{"--similarity=", "--skip-sources="},
		},
		// Verifies pre-commit.sh delegates to qa-fast-must instead of codequality-fast.
		{
			name:            "pre-commit.sh invokes qa-fast-must",
			relPath:         filepath.Join("scripts", "pre-commit.sh"),
			mustContainNorm: []string{"qa-fast-must"},
			mustNotContain:  []string{"codequality-fast"},
		},
		// Verifies pre-push.sh delegates to qa-all or qa-all-must and drops gotestutil and legacy codequality targets.
		{
			name:          "pre-push.sh delegates to qa-all without legacy targets or gotestutil",
			relPath:       filepath.Join("scripts", "pre-push.sh"),
			mustMatchNorm: []*regexp.Regexp{prePushQATargetPattern},
			mustNotContain: []string{
				"codequality-fast",
				"codequality-go",
				"codequality-web",
				"scripts/gotestutil",
			},
		},
	}

	for _, tc := range rules {
		t.Run(tc.name, func(t *testing.T) {
			rawBytes, err := os.ReadFile(filepath.Join(repoRoot, tc.relPath))
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.relPath, err)
			}

			assertFileRuleContract(t, tc, string(rawBytes))
		})
	}
}

// TestContractESLintConfigIsolation verifies that web/eslint.config.mjs does not force whole-project tsconfig loading on pure AST rules.
func TestContractESLintConfigIsolation(t *testing.T) {
	repoRoot := findContractRepoRoot(t)
	eslintPath := filepath.Join(repoRoot, "web", "eslint.config.mjs")
	rawBytes, err := os.ReadFile(eslintPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", eslintPath, err)
	}

	stripped := stripJSComments(string(rawBytes))

	if !strings.Contains(stripped, "**/*.ts") || !strings.Contains(stripped, "@typescript-eslint/no-unused-vars") {
		t.Errorf("web/eslint.config.mjs missing expected TypeScript AST rule configuration")
	}
	forbiddenTokens := []string{"tsconfig.json", "createDefaultProgram", "projectService"}
	for _, token := range forbiddenTokens {
		if strings.Contains(stripped, token) {
			t.Errorf("web/eslint.config.mjs still contains whole-project token %q", token)
		}
	}
	if jsProjectKeyPattern.MatchString(stripped) {
		t.Errorf("web/eslint.config.mjs still configures parserOptions.project")
	}
}

func assertFileRuleContract(t *testing.T, tc contractFileRule, content string) {
	t.Helper()
	normalized := normalizeShellContinuations(content)
	for _, wantSub := range tc.mustContainNorm {
		if !strings.Contains(normalized, wantSub) {
			t.Errorf("%s missing %q", tc.relPath, wantSub)
		}
	}
	for _, pat := range tc.mustMatchNorm {
		if !pat.MatchString(normalized) {
			t.Errorf("%s failed pattern %s", tc.relPath, pat.String())
		}
	}
	for _, forbidden := range tc.mustNotContain {
		if strings.Contains(content, forbidden) {
			t.Errorf("%s still contains legacy token %q", tc.relPath, forbidden)
		}
	}
	if tc.customCheck != nil {
		tc.customCheck(t, content)
	}
}

func verifyQAMakefileContract(t *testing.T, content string) {
	t.Helper()
	nonComment := makeCommentPattern.ReplaceAllString(content, "")
	if makeLangAssignPattern.MatchString(nonComment) || makeLangRefPattern.MatchString(nonComment) {
		t.Errorf("scripts/make/qa.mk must use QA_LANG and never shadow or read POSIX LANG")
	}

	phonyTargets := extractMakePhonyTargets(content)
	allNine := []string{
		"qa-fast", "qa-fast-must", "qa-fast-advisory",
		"qa-slow", "qa-slow-must", "qa-slow-advisory",
		"qa-all", "qa-all-must", "qa-all-advisory",
	}
	for _, target := range allNine {
		if _, isPhony := phonyTargets[target]; !isPhony {
			t.Errorf("scripts/make/qa.mk missing .PHONY declaration for %q", target)
		}
		if _, found := extractMakeTargetPrereqs(content, target); !found {
			t.Errorf("scripts/make/qa.mk missing rule definition for %q", target)
		}
	}

	fastTargets := []string{"qa-fast", "qa-fast-must", "qa-fast-advisory"}
	for _, target := range fastTargets {
		prereqs, found := extractMakeTargetPrereqs(content, target)
		if found && prereqs != "" {
			t.Errorf("scripts/make/qa.mk target %q has prerequisites %q, want zero build/codegen prerequisites", target, prereqs)
		}
	}
}

func verifyTestingMakefileContract(t *testing.T, content string) {
	t.Helper()
	removedTargets := []string{
		"codequality-fast",
		"codequality-decisions",
		"codequality-go",
		"codequality-web",
		"codequality-web-container",
		"codequality",
	}
	phonyTargets := extractMakePhonyTargets(content)
	for _, target := range removedTargets {
		if _, found := extractMakeTargetPrereqs(content, target); found {
			t.Errorf("scripts/make/testing.mk still defines legacy target %q", target)
		}
		if _, isPhony := phonyTargets[target]; isPhony {
			t.Errorf("scripts/make/testing.mk still lists legacy target %q in .PHONY", target)
		}
	}
	for _, kept := range []string{"codequality-ci", "post-codequality-comment"} {
		if _, found := extractMakeTargetPrereqs(content, kept); !found {
			t.Errorf("scripts/make/testing.mk missing preserved target %q", kept)
		}
	}
}

func extractMakePhonyTargets(content string) map[string]struct{} {
	joined := makeContinuationPattern.ReplaceAllString(content, " ")
	out := make(map[string]struct{})
	for line := range strings.SplitSeq(joined, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ".PHONY:") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, ".PHONY:")
		if hashIdx := strings.Index(rest, "#"); hashIdx >= 0 {
			rest = rest[:hashIdx]
		}
		for _, tok := range strings.Fields(rest) {
			out[tok] = struct{}{}
		}
	}
	return out
}

func extractMakeTargetPrereqs(content string, target string) (string, bool) {
	joined := makeContinuationPattern.ReplaceAllString(content, " ")
	prefix := target + ":"
	for line := range strings.SplitSeq(joined, "\n") {
		if strings.HasPrefix(line, "\t") {
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		after := strings.TrimPrefix(trimmed, prefix)
		if strings.HasPrefix(after, "=") {
			continue
		}
		if hashIdx := strings.Index(after, "#"); hashIdx >= 0 {
			after = after[:hashIdx]
		}
		return strings.TrimSpace(after), true
	}
	return "", false
}

func normalizeShellContinuations(content string) string {
	joined := shellContinuationPattern.ReplaceAllString(content, " ")
	return whitespaceRunPattern.ReplaceAllString(joined, " ")
}

func stripJSComments(content string) string {
	return jsTokenPattern.ReplaceAllStringFunc(content, func(token string) string {
		if strings.HasPrefix(token, "//") || strings.HasPrefix(token, "/*") {
			return ""
		}
		return token
	})
}

func findContractRepoRoot(t *testing.T) string {
	t.Helper()
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	currentDir := workDir
	for {
		if _, statErr := os.Stat(filepath.Join(currentDir, "go.mod")); statErr == nil {
			return currentDir
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			t.Fatalf("failed to locate repository go.mod from %s", workDir)
		}
		currentDir = parentDir
	}
}
