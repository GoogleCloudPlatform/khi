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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestLoadBaseSnapshotCrossFileMoveAndPeerClones(t *testing.T) {
	baseFiles := map[string]string{
		"pkg/svc/a_high.go": `package svc

func Overloaded(a, b bool) int {
	if a {
		if b {
			return 1
		}
	}
	return 0
}
`,
		"pkg/svc/z_low.go": `package svc

func Overloaded(a, b bool) int {
	return 0
}

func ZeroCogMoved(x int) int {
	return x + 1
}
`,
		"pkg/svc/z_tie.go": `package svc

func Overloaded(a, b bool) int {
	if a {
		if b {
			return 2
		}
	}
	return 0
}
`,
		"pkg/svc/clone_target.go": `package svc

func DupTarget(xs []int) int {
	s := 0
	for _, x := range xs {
		if x > 0 {
			s += x
		}
	}
	return s
}

func ExactLimitTarget(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}
`,
		"pkg/peer/unmodified_peer.go": `package peer

func DupPeer(ys []int) int {
	t := 0
	for _, y := range ys {
		if y > 0 {
			t += y
		}
	}
	return t
}

func ExactLimitPeer(ys []int) int {
	t := 0
	for _, y := range ys {
		t += y
	}
	return t
}
`,
		"pkg/testonly/old_test.go": `package testonly

func movedTestHelper() int { return 42 }
`,
		"pkg/unrelated/deleted.go": `package unrelated

func Ignored() {}
`,
	}

	th := rule.DefaultThresholds()
	th.Limits[rule.RuleCodeClone] = 4

	readCalls := 0
	snap, err := LoadBaseSnapshot(
		func(relPath string) ([]byte, bool, error) {
			readCalls++
			switch relPath {
			case "pkg/svc/gen.go":
				return []byte("// Code generated. DO NOT EDIT.\npackage svc\n"), true, nil
			case "pkg/svc/svc_test.go":
				return []byte("package svc\nimport \"testing\"\nfunc TestX(t *testing.T) { t.Run(\"sub\", func(t *testing.T) {}) }\n"), true, nil
			default:
				s, ok := baseFiles[relPath]
				return []byte(s), ok, nil
			}
		},
		th,
		[]string{"pkg/svc/new_split.go", "pkg/svc/clone_target.go", "pkg/svc/gen.go"},
		[]string{"pkg/svc/svc_test.go", "pkg/testonly/new_test.go"},
		[]string{"pkg/svc/a_high.go", "pkg/svc/z_low.go", "pkg/svc/z_tie.go", "pkg/testonly/old_test.go", "pkg/unrelated/deleted.go"},
	)
	if err != nil {
		t.Fatalf("LoadBaseSnapshot() error: %v", err)
	}

	if fm, ok := snap.FileMetrics("pkg/svc/clone_target.go"); !ok || fm.File != "pkg/svc/clone_target.go" {
		t.Errorf("FileMetrics(clone_target.go) = (%v, %v), want non-nil true", fm, ok)
	}
	if snap.HasFile("pkg/unrelated/deleted.go") {
		t.Errorf("HasFile(pkg/unrelated/deleted.go) = true, want false for untouched directory")
	}

	if _, ok := snap.LookupFunc("pkg/svc/new_split.go", "ZeroCogMoved"); !ok {
		t.Errorf("LookupFunc(ZeroCogMoved) = false, want true")
	}
	locFile, overloadedFn, ok := snap.LookupFuncLocation("pkg/svc/new_split.go", "Overloaded")
	if !ok || locFile != "pkg/svc/a_high.go" || overloadedFn.Cognitive != 3 {
		t.Errorf("LookupFuncLocation(Overloaded) = (%q, cog=%d, %v), want (pkg/svc/a_high.go, cog=3, true)", locFile, overloadedFn.Cognitive, ok)
	}
	if sameFile, _, ok := snap.LookupFuncLocation("pkg/svc/clone_target.go", "DupTarget"); !ok || sameFile != "pkg/svc/clone_target.go" {
		t.Errorf("LookupFuncLocation(DupTarget) = (%q, %v), want (pkg/svc/clone_target.go, true)", sameFile, ok)
	}
	if _, ok := snap.LookupFunc("pkg/testonly/new_test.go", "movedTestHelper"); !ok {
		t.Errorf("LookupFunc(movedTestHelper) = false, want true")
	}
	if _, ok := snap.LookupTRun("pkg/svc/svc_test.go", "TestX", "sub"); !ok {
		t.Errorf("LookupTRun() = false, want true")
	}

	if _, hasBase := snap.MeasureBaseCloneOverlap("pkg/svc/missing.go", "NoFunc", ""); hasBase {
		t.Errorf("MeasureBaseCloneOverlap(missing) hasBase = true, want false")
	}
	callsBeforeEmptyPeer := readCalls
	if overlap, hasBase := snap.MeasureBaseCloneOverlap("pkg/svc/clone_target.go", "DupTarget", ""); !hasBase || overlap != 0 {
		t.Errorf("MeasureBaseCloneOverlap(no peer) = (%d, %v), want (0, true)", overlap, hasBase)
	}
	if readCalls != callsBeforeEmptyPeer {
		t.Errorf("empty peerFile triggered readBase: readCalls %d -> %d", callsBeforeEmptyPeer, readCalls)
	}

	gotOverlap, hasBase := snap.MeasureBaseCloneOverlap("pkg/svc/clone_target.go", "DupTarget", "pkg/peer/unmodified_peer.go")
	if !hasBase || gotOverlap != 5 {
		t.Errorf("MeasureBaseCloneOverlap(DupTarget) = (%d, %v), want (5, true)", gotOverlap, hasBase)
	}
	afterFirstPeerLoad := readCalls
	if exactOverlap, ok := snap.MeasureBaseCloneOverlap("pkg/svc/clone_target.go", "ExactLimitTarget", "pkg/peer/unmodified_peer.go"); !ok || exactOverlap != 0 {
		t.Errorf("MeasureBaseCloneOverlap(ExactLimitTarget) = (%d, %v), want (0, true)", exactOverlap, ok)
	}
	if readCalls != afterFirstPeerLoad {
		t.Errorf("peerLoaded cache failed: readCalls grew from %d to %d", afterFirstPeerLoad, readCalls)
	}
}

func TestLoadBaseSnapshotNilAndError(t *testing.T) {
	th := rule.DefaultThresholds()
	nilSnap, err := LoadBaseSnapshot(nil, th, []string{"a.go"}, nil, nil)
	if err != nil || nilSnap.HasFile("a.go") {
		t.Errorf("LoadBaseSnapshot(nil) = (%v, %v)", nilSnap, err)
	}

	wantErr := errors.New("boom")
	_, err = LoadBaseSnapshot(
		func(string) ([]byte, bool, error) { return nil, false, wantErr },
		th,
		[]string{"a.go"},
		nil,
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("LoadBaseSnapshot(err) = %v, want %v", err, wantErr)
	}
}
