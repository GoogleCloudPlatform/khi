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

package astcheck

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// BaseReader reads a repository-relative file at the base revision.
type BaseReader func(relPath string) ([]byte, bool, error)

type locatedFunc struct {
	file string
	fn   FuncMetrics
}

// BaseSnapshot caches parsed AST metrics and clone blocks for touched packages at the base revision.
type BaseSnapshot struct {
	readBase    BaseReader
	thresholds  rule.Thresholds
	fileExisted map[string]bool
	fileByPath  map[string]*FileMetrics
	funcByPkg   map[string]locatedFunc
	trunByKey   map[string]TRunMetrics
	clonesByKey map[string]FuncCloneBlock
	peerLoaded  map[string]bool
	baseBlocks  []FuncCloneBlock
}

// LoadBaseSnapshot parses base-revision AST metrics for prodFiles, testFiles, and deletedGoFiles in touched packages.
func LoadBaseSnapshot(
	readBase BaseReader,
	thresholds rule.Thresholds,
	prodFiles []string,
	testFiles []string,
	deletedGoFiles []string,
) (*BaseSnapshot, error) {
	snap := &BaseSnapshot{
		readBase:    readBase,
		thresholds:  thresholds,
		fileExisted: make(map[string]bool),
		fileByPath:  make(map[string]*FileMetrics),
		funcByPkg:   make(map[string]locatedFunc),
		trunByKey:   make(map[string]TRunMetrics),
		clonesByKey: make(map[string]FuncCloneBlock),
		peerLoaded:  make(map[string]bool),
	}
	if readBase == nil {
		return snap, nil
	}

	touchedDirs := collectDirectories(prodFiles, testFiles)
	candidates := collectBaseCandidateFiles(prodFiles, testFiles, deletedGoFiles, touchedDirs)
	for _, relPath := range candidates {
		if err := snap.ingestBaseFile(relPath); err != nil {
			return nil, err
		}
	}
	return snap, nil
}

func (s *BaseSnapshot) ingestBaseFile(relPath string) error {
	s.peerLoaded[relPath] = true
	src, exists, err := s.readBase(relPath)
	if err != nil || !exists {
		return err
	}
	s.fileExisted[relPath] = true
	fm, err := AnalyzeFile(relPath, src, s.thresholds.MinParamsForFlagCheck)
	if err != nil || fm == nil {
		return nil
	}
	s.recordFileMetrics(relPath, src, fm)
	return nil
}

func (s *BaseSnapshot) recordFileMetrics(relPath string, src []byte, fm *FileMetrics) {
	s.fileByPath[relPath] = fm
	dir := path.Dir(relPath)
	for _, fn := range fm.Funcs {
		pkgKey := fmt.Sprintf("%s|%v|%s", dir, fm.IsTest, fn.Key)
		if existing, ok := s.funcByPkg[pkgKey]; !ok || fn.Cognitive > existing.fn.Cognitive {
			s.funcByPkg[pkgKey] = locatedFunc{file: relPath, fn: fn}
		}
	}
	for _, tr := range fm.TRuns {
		trKey := fmt.Sprintf("%s|%s|%s", relPath, tr.ParentFunc, tr.SubtestLabel)
		s.trunByKey[trKey] = tr
	}
	if !fm.IsTest {
		blocks := ExtractCloneBlocks(relPath, src)
		for _, blk := range blocks {
			s.clonesByKey[relPath+"|"+blk.FuncKey] = blk
		}
		s.baseBlocks = append(s.baseBlocks, blocks...)
	}
}

// HasFile reports whether relPath existed at the base revision.
func (s *BaseSnapshot) HasFile(relPath string) bool {
	return s.fileExisted[relPath]
}

// FileMetrics returns parsed base FileMetrics for relPath when available.
func (s *BaseSnapshot) FileMetrics(relPath string) (*FileMetrics, bool) {
	fm, ok := s.fileByPath[relPath]
	return fm, ok
}

// LookupFunc looks up funcKey at base in relPath first, then across the same package directory.
func (s *BaseSnapshot) LookupFunc(relPath string, funcKey string) (FuncMetrics, bool) {
	_, fn, ok := s.LookupFuncLocation(relPath, funcKey)
	return fn, ok
}

// LookupFuncLocation returns the base file path and FuncMetrics for funcKey at base.
func (s *BaseSnapshot) LookupFuncLocation(relPath string, funcKey string) (string, FuncMetrics, bool) {
	if baseFile := s.fileByPath[relPath]; baseFile != nil {
		for _, bfn := range baseFile.Funcs {
			if bfn.Key == funcKey {
				return relPath, bfn, true
			}
		}
	}
	isTest := strings.HasSuffix(relPath, "_test.go")
	pkgKey := fmt.Sprintf("%s|%v|%s", path.Dir(relPath), isTest, funcKey)
	loc, ok := s.funcByPkg[pkgKey]
	return loc.file, loc.fn, ok
}

// LookupTRun looks up a subtest closure at base by file, parent function, and label.
func (s *BaseSnapshot) LookupTRun(relPath string, parentFunc string, subtestLabel string) (TRunMetrics, bool) {
	trKey := fmt.Sprintf("%s|%s|%s", relPath, parentFunc, subtestLabel)
	tr, ok := s.trunByKey[trKey]
	return tr, ok
}

// MeasureBaseCloneOverlap measures the longest base-revision clone span for (relPath, funcKey), lazily loading peerFile.
func (s *BaseSnapshot) MeasureBaseCloneOverlap(relPath string, funcKey string, peerFile string) (int, bool) {
	baseBlk, hasBase := s.clonesByKey[relPath+"|"+funcKey]
	if !hasBase {
		return 0, false
	}
	if peerFile != "" && !s.peerLoaded[peerFile] {
		_ = s.ingestBaseFile(peerFile)
	}
	minWindow := s.thresholds.Limit(rule.RuleCodeClone) + 1
	idx := NewCloneIndex(minWindow, s.baseBlocks)
	baseMatch, _ := idx.FindMaxClone(baseBlk)
	return baseMatch.Statements, true
}

func collectDirectories(prodFiles []string, testFiles []string) map[string]struct{} {
	dirs := make(map[string]struct{})
	for _, f := range slices.Concat(prodFiles, testFiles) {
		dirs[path.Dir(f)] = struct{}{}
	}
	return dirs
}

func collectBaseCandidateFiles(prodFiles []string, testFiles []string, deleted []string, touchedDirs map[string]struct{}) []string {
	seen := make(map[string]struct{})
	for _, f := range prodFiles {
		seen[f] = struct{}{}
	}
	for _, f := range testFiles {
		seen[f] = struct{}{}
	}
	for _, f := range deleted {
		if _, ok := touchedDirs[path.Dir(f)]; ok {
			seen[f] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}
