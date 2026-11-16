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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func TestRenderTSSuperMutantAndRuntimeSemantics(t *testing.T) {
	if got := string(RenderTSSuperMutantSource([]byte("const x = 1;"), nil)); got != "const x = 1;" {
		t.Errorf("RenderTSSuperMutantSource(empty) = %q, want unchanged source", got)
	}

	m1 := &TSMutant{ID: 1}
	m2 := &TSMutant{ID: 2}

	t.Run("skips duplicate span at excludeIdx zero", func(t *testing.T) {
		dupSites := []TSMutationSite{
			{Kind: "cond", StartOffset: 0, EndOffset: 5, Mutants: []*TSMutant{m1, m2}},
			{Kind: "cond", StartOffset: 0, EndOffset: 5, Mutants: []*TSMutant{m1, m2}},
		}
		got := string(RenderTSSuperMutantSource([]byte("a < b"), dupSites))
		want := "// @ts-nocheck\n(globalThis.__khiMut.cond(1, 2, Boolean(a < b)))"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("duplicate root span render mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("renders all binary unary literal statement and fallback site kinds", func(t *testing.T) {
		src, sites, want := buildComprehensiveRenderFixture(m1, m2)
		got := string(RenderTSSuperMutantSource(src, sites))
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("RenderTSSuperMutantSource() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("generates exact capacity buffers and executes VM runtime probes", func(t *testing.T) {
		assertRuntimeSourceCapacities(t, 4, 3, []string{
			"var topLevelSeen = new Uint8Array(5);",
			"if (pId >= 4)",
			"if (id === 0 || id >= 5)",
			"currentSpecMutantSeen = new Uint8Array(5);",
			"currentSpecSlotSeen = new Uint8Array(15);",
			"currentSpecProbeSlotSeen = new Uint8Array(12);",
		})
		assertRuntimeSourceCapacities(t, 0, 0, []string{
			"var topLevelSeen = new Uint8Array(2);",
			"if (pId >= 2)",
			"currentSpecSlotSeen = new Uint8Array(6);",
			"currentSpecProbeSlotSeen = new Uint8Array(6);",
		})
		verifyBrowserRuntimeWithNodeVM(t)
	})
}

func buildComprehensiveRenderFixture(m1 *TSMutant, m2 *TSMutant) ([]byte, []TSMutationSite, string) {
	src := []byte("a < b || a <= b && a > b && a >= b && a === b && x ?? y; p + q; r + s; t - u; v * w; x / y; !z; true; false; noop; call();")
	sites := []TSMutationSite{
		{Kind: "ordered_cmp", Op: "<", StartOffset: 0, EndOffset: 5, LeftStart: 0, LeftEnd: 1, RightStart: 4, RightEnd: 5, Mutants: []*TSMutant{m1, m2}},
		{Kind: "ordered_cmp", Op: "<=", StartOffset: 9, EndOffset: 15, LeftStart: 9, LeftEnd: 10, RightStart: 14, RightEnd: 15, Mutants: []*TSMutant{m1, m2}},
		{Kind: "logical", Op: "||", StartOffset: 0, EndOffset: 15, LeftStart: 0, LeftEnd: 5, RightStart: 9, RightEnd: 15, Mutants: []*TSMutant{m1}},
		{Kind: "ordered_cmp", Op: ">", StartOffset: 19, EndOffset: 24, LeftStart: 19, LeftEnd: 20, RightStart: 23, RightEnd: 24, Mutants: []*TSMutant{m1, m2}},
		{Kind: "logical", Op: "&&", StartOffset: 19, EndOffset: 34, LeftStart: 19, LeftEnd: 24, RightStart: 28, RightEnd: 34, Mutants: []*TSMutant{m1}},
		{Kind: "ordered_cmp", Op: ">=", StartOffset: 28, EndOffset: 34, LeftStart: 28, LeftEnd: 29, RightStart: 33, RightEnd: 34, Mutants: []*TSMutant{m1, m2}},
		{Kind: "eq_cmp", Op: "===", StartOffset: 38, EndOffset: 45, LeftStart: 38, LeftEnd: 39, RightStart: 44, RightEnd: 45, Mutants: []*TSMutant{m1}},
		{Kind: "nullish", Op: "??", StartOffset: 49, EndOffset: 55, LeftStart: 49, LeftEnd: 50, RightStart: 54, RightEnd: 55, Mutants: []*TSMutant{m1, m2}},
		{Kind: "arith", Op: "+", AddKeepRight: false, StartOffset: 57, EndOffset: 62, LeftStart: 57, LeftEnd: 58, RightStart: 61, RightEnd: 62, Mutants: []*TSMutant{m1}},
		{Kind: "arith", Op: "+", AddKeepRight: true, StartOffset: 64, EndOffset: 69, LeftStart: 64, LeftEnd: 65, RightStart: 68, RightEnd: 69, Mutants: []*TSMutant{m1}},
		{Kind: "arith", Op: "-", StartOffset: 71, EndOffset: 76, LeftStart: 71, LeftEnd: 72, RightStart: 75, RightEnd: 76, Mutants: []*TSMutant{m1}},
		{Kind: "arith", Op: "*", StartOffset: 78, EndOffset: 83, LeftStart: 78, LeftEnd: 79, RightStart: 82, RightEnd: 83, Mutants: []*TSMutant{m1}},
		{Kind: "arith", Op: "/", StartOffset: 85, EndOffset: 90, LeftStart: 85, LeftEnd: 86, RightStart: 89, RightEnd: 90, Mutants: []*TSMutant{m1}},
		{Kind: "unary_not", StartOffset: 92, EndOffset: 94, LeftStart: 93, LeftEnd: 94, Mutants: []*TSMutant{m1}},
		{Kind: "bool_lit", BoolVal: true, StartOffset: 96, EndOffset: 100, Mutants: []*TSMutant{m1}},
		{Kind: "bool_lit", BoolVal: false, StartOffset: 102, EndOffset: 107, Mutants: []*TSMutant{m1}},
		{Kind: "unknown", StartOffset: 109, EndOffset: 113, Mutants: []*TSMutant{m1}},
		{Kind: "stmt_skip", StartOffset: 115, EndOffset: 122, Mutants: []*TSMutant{m1}},
	}
	want := "// @ts-nocheck\n" +
		"(globalThis.__khiMut.or(1, () => ((globalThis.__khiMut.lt(1, 2, (a), (b)))), () => ((globalThis.__khiMut.le(1, 2, (a), (b)))))) && " +
		"(globalThis.__khiMut.and(1, () => ((globalThis.__khiMut.gt(1, 2, (a), (b)))), () => ((globalThis.__khiMut.ge(1, 2, (a), (b)))))) && " +
		"(globalThis.__khiMut.flip(1, (a) === (b))) && " +
		"(globalThis.__khiMut.coal(1, 2, () => (x), () => (y))); " +
		"(globalThis.__khiMut.addL(1, (p), (q))); " +
		"(globalThis.__khiMut.addR(1, (r), (s))); " +
		"(globalThis.__khiMut.sub(1, (t), (u))); " +
		"(globalThis.__khiMut.mul(1, (v), (w))); " +
		"(globalThis.__khiMut.div(1, (x), (y))); " +
		"(globalThis.__khiMut.flip(1, !(z))); " +
		"(globalThis.__khiMut.flip(1, true)); " +
		"(globalThis.__khiMut.flip(1, false)); " +
		"noop; " +
		"if (!globalThis.__khiMut.active(1)) { call(); }"
	return src, sites, want
}

func assertRuntimeSourceCapacities(t *testing.T, maxMutantID int, maxProbeID int, wantSnippets []string) {
	t.Helper()
	script := string(GenerateTSBrowserRuntimeSource(maxMutantID, maxProbeID))
	for _, snippet := range wantSnippets {
		if !strings.Contains(script, snippet) {
			t.Errorf("GenerateTSBrowserRuntimeSource(%d, %d) missing %q", maxMutantID, maxProbeID, snippet)
		}
	}
}

func verifyBrowserRuntimeWithNodeVM(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	runtimeFile := filepath.Join(tmpDir, tsBrowserRuntimeFileName)
	if err := os.WriteFile(runtimeFile, GenerateTSBrowserRuntimeSource(5, 3), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	driverFile := filepath.Join(tmpDir, "driver.js")
	driverSrc := `
const fs = require('fs');
const vm = require('vm');
const runtimeCode = fs.readFileSync(process.argv[2], 'utf8');

function runWithArgs(args, fn) {
  let reported = null;
  let registeredReporter = null;
  const sandbox = {
    __karma__: {
      config: { args },
      info: (payload) => { reported = payload; },
      start: () => {},
    },
    jasmine: {
      getEnv: () => ({
        addReporter: (rep) => { registeredReporter = rep; },
      }),
    },
  };
  sandbox.globalThis = sandbox;
  vm.createContext(sandbox);
  vm.runInContext(runtimeCode, sandbox);
  sandbox.__karma__.start();
  const value = fn(sandbox.__khiMut, registeredReporter);
  return { value, reported };
}

const base = runWithArgs(['--khi-mutant=0'], (m, rep) => {
  const topHit = m.lt(1, 2, 5n, 10n);
  rep.specStarted({ id: 'spec0', fullName: 'SampleSuite handles values' });
  const andVal = m.and(3, () => 'truthy', () => 'rhs');
  const coalVal = m.coal(4, 5, () => null, () => 'fallback');
  const probeVal = m.lt(-3, -3, 1, 10);
  rep.specDone({ id: 'spec0', fullName: 'SampleSuite handles values', status: 'passed' });
  rep.jasmineDone();
  return { topHit, andVal, coalVal, probeVal };
});

let rhsCalled = false;
const mutAnd = runWithArgs(['--khi-mutant=3'], (m) =>
  m.and(3, () => 'leftVal', () => { rhsCalled = true; return 'rightVal'; })
);
const mutCoalLeft = runWithArgs(['--khi-mutant=4'], (m) =>
  m.coal(4, 5, () => null, () => 'fallback')
);

process.stdout.write(JSON.stringify({
  baseValue: base.value,
  specHits: base.reported.khiMutantReport.specHits,
  specOrder: base.reported.khiMutantReport.specOrder,
  specPolarities: base.reported.khiMutantReport.specPolarities,
  topLevelHits: base.reported.khiMutantReport.topLevelHits,
  mutAnd: mutAnd.value,
  rhsCalled,
  mutCoalLeft: mutCoalLeft.value,
}));
`
	if err := os.WriteFile(driverFile, []byte(driverSrc), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	outBytes, err := gotestutil.RunCommandOutput(context.Background(), tmpDir, "node", driverFile, runtimeFile)
	if err != nil {
		t.Fatalf("runtime driver failed: %v", err)
	}

	var got struct {
		SpecHits       map[string][]int              `json:"specHits"`
		SpecOrder      []redundancy.TSSpecOrderEntry `json:"specOrder"`
		SpecPolarities map[string][]string           `json:"specPolarities"`
		TopLevelHits   []int                         `json:"topLevelHits"`
		MutAnd         string                        `json:"mutAnd"`
		RhsCalled      bool                          `json:"rhsCalled"`
		MutCoalLeft    *string                       `json:"mutCoalLeft"`
	}
	if err := json.Unmarshal(outBytes, &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if diff := cmp.Diff([]int{1, 2}, got.TopLevelHits); diff != "" {
		t.Errorf("topLevelHits mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string][]string{"spec0": {"3:T", "4:F", "5:F", "P3:T"}}, got.SpecPolarities); diff != "" {
		t.Errorf("specPolarities mismatch (-want +got):\n%s", diff)
	}
	if got.MutAnd != "leftVal" || got.RhsCalled || got.MutCoalLeft != nil {
		t.Errorf("unexpected active mutant behavior: mutAnd=%q rhsCalled=%v mutCoalLeft=%v", got.MutAnd, got.RhsCalled, got.MutCoalLeft)
	}
}
