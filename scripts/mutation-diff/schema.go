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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const runtimeHelperFileName = "zzz_khi_mutant_runtime.go"

// OverlayFileSpec holds a single mutated Go file to materialize into a temporary overlay directory.
type OverlayFileSpec struct {
	RelFile string
	Source  []byte
	Sites   []MutationSite
}

type goOverlayJSON struct {
	Replace map[string]string `json:"Replace"`
}

// GenerateRuntimeSource returns the Go source for zzz_khi_mutant_runtime.go for package pkgName.
func GenerateRuntimeSource(pkgName string, maxMutantID int) []byte {
	seenCap := max(maxMutantID+1, 2)
	content := fmt.Sprintf(`package %s

import (
	"cmp"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

type __khiMutNumeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

type __khiMutAddable interface {
	__khiMutNumeric | ~string
}

var __khiMutSelected = sync.OnceValue(func() int {
	v, err := strconv.Atoi(os.Getenv("KHI_MUTANT_ID"))
	if err != nil {
		return 0
	}
	return v
})

var __khiMutTracePath = sync.OnceValue(func() string {
	return os.Getenv("KHI_MUTANT_TRACE_FILE")
})

var __khiMutSeen [%d]atomic.Bool

func __khiMutRecord(id int) {
	if id <= 0 || id >= len(__khiMutSeen) {
		return
	}
	if __khiMutSeen[id].Load() || __khiMutSeen[id].Swap(true) {
		return
	}
	p := __khiMutTracePath()
	if p == "" {
		return
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(strconv.Itoa(id) + "\n")
	_ = f.Close()
}

func __khiMutActive(id int) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(id)
		return false
	}
	return sel == id
}

func __khiMutFlip(id int, v bool) bool {
	if __khiMutActive(id) {
		return !v
	}
	return v
}

func __khiMutCond(idTrue, idFalse int, cond bool) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(idTrue)
		__khiMutRecord(idFalse)
		return cond
	}
	if sel == idTrue {
		return true
	}
	if sel == idFalse {
		return false
	}
	return cond
}

func __khiMutAnd(id int, lhs, rhs func() bool) bool {
	if __khiMutActive(id) {
		return lhs() || rhs()
	}
	return lhs() && rhs()
}

func __khiMutOr(id int, lhs, rhs func() bool) bool {
	if __khiMutActive(id) {
		return lhs() && rhs()
	}
	return lhs() || rhs()
}

func __khiMutLT[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(id1)
		__khiMutRecord(id2)
		return a < b
	}
	if sel == id1 {
		return a <= b
	}
	if sel == id2 {
		return a >= b
	}
	return a < b
}

func __khiMutLE[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(id1)
		__khiMutRecord(id2)
		return a <= b
	}
	if sel == id1 {
		return a < b
	}
	if sel == id2 {
		return a > b
	}
	return a <= b
}

func __khiMutGT[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(id1)
		__khiMutRecord(id2)
		return a > b
	}
	if sel == id1 {
		return a >= b
	}
	if sel == id2 {
		return a <= b
	}
	return a > b
}

func __khiMutGE[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecord(id1)
		__khiMutRecord(id2)
		return a >= b
	}
	if sel == id1 {
		return a > b
	}
	if sel == id2 {
		return a < b
	}
	return a >= b
}

func __khiMutAddL[T __khiMutAddable](id int, a, b T) T {
	if __khiMutActive(id) {
		return a
	}
	return a + b
}

func __khiMutAddR[T __khiMutAddable](id int, a, b T) T {
	if __khiMutActive(id) {
		return b
	}
	return a + b
}

func __khiMutSub[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a + b
	}
	return a - b
}

func __khiMutMul[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a / b
	}
	return a * b
}

func __khiMutDiv[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a * b
	}
	return a / b
}
`, pkgName, seenCap)
	return []byte(content)
}

// WritePackageOverlay materializes Super-Mutant Go sources, the runtime helper, and overlay.json inside workDir.
func WritePackageOverlay(
	repoRoot string,
	pkgRelDir string,
	pkgName string,
	files []OverlayFileSpec,
	maxMutantID int,
	workDir string,
) (string, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo root %s: %w", repoRoot, err)
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve overlay workdir %s: %w", workDir, err)
	}

	replace := make(map[string]string, len(files)+1)
	srcStaging := filepath.Join(absWork, "src")

	for _, f := range files {
		if len(f.Sites) == 0 {
			continue
		}
		rendered := RenderSuperMutantSource(f.Source, f.Sites)
		dstPath := filepath.Join(srcStaging, filepath.FromSlash(f.RelFile))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return "", fmt.Errorf("failed to create overlay dir for %s: %w", dstPath, err)
		}
		if err := os.WriteFile(dstPath, rendered, 0o600); err != nil {
			return "", fmt.Errorf("failed to write mutated file %s: %w", dstPath, err)
		}
		origAbs := filepath.Join(absRoot, filepath.FromSlash(f.RelFile))
		replace[origAbs] = dstPath
	}

	runtimeBytes := GenerateRuntimeSource(pkgName, maxMutantID)
	runtimeDst := filepath.Join(srcStaging, filepath.FromSlash(pkgRelDir), runtimeHelperFileName)
	if err := os.MkdirAll(filepath.Dir(runtimeDst), 0o755); err != nil {
		return "", fmt.Errorf("failed to create runtime helper dir: %w", err)
	}
	if err := os.WriteFile(runtimeDst, runtimeBytes, 0o600); err != nil {
		return "", fmt.Errorf("failed to write runtime helper %s: %w", runtimeDst, err)
	}
	origRuntimeAbs := filepath.Join(absRoot, filepath.FromSlash(pkgRelDir), runtimeHelperFileName)
	replace[origRuntimeAbs] = runtimeDst

	overlayBytes, err := json.MarshalIndent(goOverlayJSON{Replace: replace}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal overlay.json: %w", err)
	}
	overlayPath := filepath.Join(absWork, "overlay.json")
	if err := os.WriteFile(overlayPath, overlayBytes, 0o600); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", overlayPath, err)
	}
	return overlayPath, nil
}

// ParseTraceFile reads unique reached mutant IDs written to tracePath during the baseline run.
func ParseTraceFile(tracePath string) (map[int]struct{}, error) {
	reached := make(map[int]struct{})
	f, err := os.Open(tracePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return reached, nil
		}
		return nil, fmt.Errorf("failed to open mutant trace file %s: %w", tracePath, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		txt := strings.TrimSpace(sc.Text())
		if txt == "" {
			continue
		}
		id, err := strconv.Atoi(txt)
		if err != nil {
			return nil, fmt.Errorf("invalid mutant id %q in trace file: %w", txt, err)
		}
		if id > 0 {
			reached[id] = struct{}{}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed reading mutant trace file %s: %w", tracePath, err)
	}
	return reached, nil
}
