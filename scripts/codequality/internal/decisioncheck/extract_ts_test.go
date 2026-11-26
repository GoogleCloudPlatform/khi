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

package decisioncheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/gotestutil"
)

func TestExtractChangedTSDecls(t *testing.T) {
	webDir := skipIfTypeScriptUnavailable(t)

	sampleTS := `// Copyright 2026 Example Contributors
import { Injectable } from '@angular/core'; // trailing import comment
export interface AfterImport { ok: boolean; }
{
  const blockScoped = 1;
  function hiddenInBlock() {}
} // trailing block comment
export enum Mode {
  Fast = 'fast',
  Slow = 'slow',
} // trailing enum comment
export const enum FlagBit {
  None = 0,
  Read = 1,
}

/** Docs a config interface. */
export interface SampleConfig {
  timeoutMs: number;
}

/**
 * Multiline record type.
 */
export type RecordEntry = {
  id: string;
  enabled: boolean;
};

// Separated by blank line.

// First paragraph not attached.
//
// Computes sum across items.
// Ignores negative inputs.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function computeSum<T extends number>(items: T[]): number {
  return items.length;
} // trailing function comment

export const runArrow = (value: number): number => {
  return value + 1;
};

const runFnExpr = function (value: number): number {
  return value;
};

const exprArrow = (n: number) => n + 1;
export const wrappedMemo = memoize((() => 1) as () => number);
export const multiA = () => 1, multiB = () => 2;
const plainObj = { count: 1, label: 'ok' };
const [ignoredArrow] = [() => 1];

/** Service managing sample items. */
@ng.Component({
  selector: 'sample-panel',
  standalone: true,
})
export class SamplePanel extends createPanelBase({ debug: true }) {
  readonly plainSignal = signal(0); // trailing signal comment
  readonly doubledSignal = computed(() => {
    return this.plainSignal() * 2;
  });

  constructor(private readonly id: string) {}

  // Handles click events.
  @HostListener('click')
  public async handleClick<T extends Event>(ev: T): Promise<void> {
    await Promise.resolve(ev);
  }

  readonly arrowHandler = (x: number): number => {
    return x + 1;
  }; // trailing arrow comment

  /** Runs side effect on state. */
  @TrackEffect()
  readonly stateEffect = effect(function logState() {
    return;
  });
  get current(): number {
    return 1;
  }
  set current(next: number) {}
  abstract render(): void;
}

@Injectable
export class PlainWorker { // trailing class brace comment
  work<U, V extends Array<U>>(items: V): U {
    return items[0];
  }

  readonly label: string;
}
`

	widgetTS := `export class Widget {
  // Alpha doc.
  alpha(): number {
    return 1;
  }

  beta(): number {
    return 2;
  }
}

export function untouchedHelper(): void {}

export function touchedAtEnd(): number {
  return 3;
}

export class HeaderTouched {
  ping(): number {
    return 1;
  }
}

export class BodyTouched {
  pong(): number {
    return 2;
  }
  readonly extra = 1;
}
`

	testCases := []struct {
		name         string
		files        []BundledFile
		changedLines map[string]map[int]struct{}
		want         []ExtractedItem
	}{
		// Extracts all top-level TS declarations, classes, members, decorators, directives, and comment spans while skipping anonymous and bodyless declarations.
		{
			name: "full mode extracts declarations, decorators, members, and comment spans",
			files: []BundledFile{
				{RelPath: "web/src/app/empty.ts", Content: []byte(``)},
				{RelPath: "web/src/app/anon_class.ts", Content: []byte(`export default class { anon() {} }`)},
				{RelPath: "web/src/app/anon_fn.ts", Content: []byte(`export default function () {}`)},
				{RelPath: "web/src/app/anon_iface.ts", Content: []byte(`interface { anonProp: string; }`)},
				{RelPath: "web/src/app/declare_fn.ts", Content: []byte(`declare function ambient(x: number): void;`)},
				{RelPath: "web/src/app/directives.ts", Content: []byte(`// @ts-expect-error
// prettier-ignore
// CODEQUALITY_ALLOW(func_cognitive, equivalent): fixture
/*   */
export function afterDirectives() {}`)},
				{RelPath: "web/src/app/sample.ts", Content: []byte(sampleTS)},
			},
			changedLines: nil,
			want: []ExtractedItem{
				{Target: "web/app", File: "web/src/app/directives.ts", Line: 5, StartLine: 5, EndLine: 5, Name: "afterDirectives", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 3, StartLine: 3, EndLine: 3, Name: "AfterImport", Kind: "interface"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 8, StartLine: 8, EndLine: 11, Name: "Mode", Kind: "enum"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 12, StartLine: 12, EndLine: 15, Name: "FlagBit", Kind: "enum"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 18, StartLine: 17, EndLine: 20, Name: "SampleConfig", Kind: "interface", Doc: "Docs a config interface."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 25, StartLine: 22, EndLine: 28, Name: "RecordEntry", Kind: "type", Doc: "Multiline record type."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 37, StartLine: 34, EndLine: 39, Name: "computeSum", Kind: "function", Doc: "Computes sum across items.\nIgnores negative inputs."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 41, StartLine: 41, EndLine: 43, Name: "runArrow", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 45, StartLine: 45, EndLine: 47, Name: "runFnExpr", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 49, StartLine: 49, EndLine: 49, Name: "exprArrow", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 50, StartLine: 50, EndLine: 50, Name: "wrappedMemo", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 51, StartLine: 51, EndLine: 51, Name: "multiA", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 51, StartLine: 51, EndLine: 51, Name: "multiB", Kind: "function"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 60, StartLine: 55, EndLine: 88, Name: "SamplePanel", Kind: "class", Doc: "Service managing sample items."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 62, StartLine: 62, EndLine: 64, Name: "SamplePanel.doubledSignal", Kind: "method"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 66, StartLine: 66, EndLine: 66, Name: "SamplePanel.constructor", Kind: "method"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 70, StartLine: 68, EndLine: 72, Name: "SamplePanel.handleClick", Kind: "method", Doc: "Handles click events."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 74, StartLine: 74, EndLine: 76, Name: "SamplePanel.arrowHandler", Kind: "method"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 80, StartLine: 78, EndLine: 82, Name: "SamplePanel.stateEffect", Kind: "method", Doc: "Runs side effect on state."},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 83, StartLine: 83, EndLine: 85, Name: "SamplePanel.get_current", Kind: "method"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 86, StartLine: 86, EndLine: 86, Name: "SamplePanel.set_current", Kind: "method"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 91, StartLine: 90, EndLine: 97, Name: "PlainWorker", Kind: "class"},
				{Target: "web/app", File: "web/src/app/sample.ts", Line: 92, StartLine: 92, EndLine: 94, Name: "PlainWorker.work", Kind: "method"},
			},
		},
		// Emits members on exact StartLine/EndLine hits without emitting their enclosing class, emits classes when non-member lines change, and skips untouched files.
		{
			name: "differential mode handles class-vs-member overlap and skips untouched files",
			files: []BundledFile{
				{RelPath: "web/src/app/widget.ts", Content: []byte(widgetTS)},
				{RelPath: "web/src/app/untouched.ts", Content: []byte(`export function ignoredFromUntouchedFile(): void {}`)},
			},
			changedLines: map[string]map[int]struct{}{
				"web/src/app/widget.ts": {
					2:  {},
					9:  {},
					16: {},
					18: {},
					28: {},
				},
			},
			want: []ExtractedItem{
				{Target: "web/app", File: "web/src/app/widget.ts", Line: 3, StartLine: 2, EndLine: 5, Name: "Widget.alpha", Kind: "method", Doc: "Alpha doc."},
				{Target: "web/app", File: "web/src/app/widget.ts", Line: 7, StartLine: 7, EndLine: 9, Name: "Widget.beta", Kind: "method"},
				{Target: "web/app", File: "web/src/app/widget.ts", Line: 14, StartLine: 14, EndLine: 16, Name: "touchedAtEnd", Kind: "function"},
				{Target: "web/app", File: "web/src/app/widget.ts", Line: 18, StartLine: 18, EndLine: 22, Name: "HeaderTouched", Kind: "class"},
				{Target: "web/app", File: "web/src/app/widget.ts", Line: 24, StartLine: 24, EndLine: 29, Name: "BodyTouched", Kind: "class"},
			},
		},
		// Excludes spec, ambient declaration, Storybook, non-TS, and generated files while keeping valid production TS files.
		{
			name: "filters out spec, declaration, stories, non-TS, and generated files",
			files: []BundledFile{
				{RelPath: "web/src/app/comp.spec.ts", Content: []byte(`export class SpecHelper { run() {} }`)},
				{RelPath: "web/src/app/types.d.ts", Content: []byte(`export interface AmbientConfig { enabled: boolean; }`)},
				{RelPath: "web/src/app/comp.stories.ts", Content: []byte(`export function primaryStory() { return {}; }`)},
				{RelPath: "web/src/app/notes.md", Content: []byte(`export function markdownSnippet() {}`)},
				{RelPath: "web/src/app/gen1.ts", Content: []byte(`// @generated
export function g1() {}`)},
				{RelPath: "web/src/app/gen2.ts", Content: []byte(`// Code generated by tool. DO NOT EDIT.
export function g2() {}`)},
				{RelPath: "web/src/app/gen3.ts", Content: []byte(`// AUTO-GENERATED
export function g3() {}`)},
				{RelPath: "web/src/app/comp.ts", Content: []byte(`export function prodHelper() { return 1; }`)},
			},
			changedLines: nil,
			want: []ExtractedItem{
				{Target: "web/app", File: "web/src/app/comp.ts", Line: 1, StartLine: 1, EndLine: 1, Name: "prodHelper", Kind: "function"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractChangedTSDecls(context.Background(), "web/app", webDir, tc.files, tc.changedLines)

			if err != nil {
				t.Fatalf("ExtractChangedTSDecls() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractChangedTSDecls() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestExtractChangedTSFuncs verifies that ExtractChangedTSFuncs excludes non-callable declarations and untouched functions while keeping touched functions and class methods.
func TestExtractChangedTSFuncs(t *testing.T) {
	skipIfTypeScriptUnavailable(t)
	src := `export interface Config {
  enabled: boolean;
}

export type Mode = 'a' | 'b';

export class Service {
  readonly count = signal(0);
  readonly doubled = computed(() => {
    return this.count() * 2;
  });

  run(): number {
    return 1;
  }
}

export function topAction(): void {}

export const arrowAction = (): number => {
  return 2;
};

export function untouchedAction(): void {}
`
	files := []BundledFile{{RelPath: "web/src/app/funcs.ts", Content: []byte(src)}}
	changedLines := map[string]map[int]struct{}{
		"web/src/app/funcs.ts": {
			2:  {},
			5:  {},
			7:  {},
			8:  {},
			10: {},
			14: {},
			18: {},
			21: {},
		},
	}

	got, err := ExtractChangedTSFuncs(context.Background(), "web/app", "", files, changedLines)

	if err != nil {
		t.Fatalf("ExtractChangedTSFuncs() unexpected error: %v", err)
	}
	want := []ExtractedItem{
		{Target: "web/app", File: "web/src/app/funcs.ts", Line: 9, StartLine: 9, EndLine: 11, Name: "Service.doubled", Kind: "method"},
		{Target: "web/app", File: "web/src/app/funcs.ts", Line: 13, StartLine: 13, EndLine: 15, Name: "Service.run", Kind: "method"},
		{Target: "web/app", File: "web/src/app/funcs.ts", Line: 18, StartLine: 18, EndLine: 18, Name: "topAction", Kind: "function"},
		{Target: "web/app", File: "web/src/app/funcs.ts", Line: 20, StartLine: 20, EndLine: 22, Name: "arrowAction", Kind: "function"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ExtractChangedTSFuncs() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractChangedTSSpecs(t *testing.T) {
	webDir := skipIfTypeScriptUnavailable(t)

	suitesSrc := `it('token zero spec', () => {});
describe('Outer', () => {
  // Explains focused spec.
  fdescribe('Inner', () => {
    fit('focused spec', () => {
      it('nested call ignored', () => {});
    });
  });
  xdescribe('Skipped', () => {
    xit('skipped spec', () => {});
  });
});
describe(` + "`TplSuite`" + `, () => {
  it(` + "`tpl spec`" + `, () => {});
});
`

	commentsSrc := `helper.it('member call ignored', () => {});
expect('actual').toBe('actual');
it(dynamicTitle, () => {});
const x = 1; // semi boundary comment
it('after semi', () => {});
// Suite comment not stolen by inline spec.
describe('InlineSuite', () => it('inline spec', () => {}));
// Paren-wrapped spec preserves comment.
(it('paren wrapped', () => {}));
/**
 * Block comment on spec.
 */
it('after block doc', () => {});
`

	diffSrc := `it('untouched one', () => {});

// Explains touched spec.
it('touched two', () => {});

it('untouched three', () => {});
`

	testCases := []struct {
		name         string
		targetID     string
		files        []BundledFile
		changedLines map[string]map[int]struct{}
		want         []ExtractedItem
	}{
		// Extracts top-level and nested describe/fdescribe/xdescribe specs with comments and ignores nested, member, and dynamic calls.
		{
			name:     "extracts suite hierarchy, comments, inline suites, and paren-wrapped specs",
			targetID: "web/src/app/demo.spec.ts",
			files: []BundledFile{
				{RelPath: "web/src/app/demo.spec.ts", Content: []byte(suitesSrc)},
				{RelPath: "web/src/app/edges.spec.ts", Content: []byte(commentsSrc)},
			},
			want: []ExtractedItem{
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/demo.spec.ts", Line: 1, StartLine: 1, EndLine: 1, Name: `"token zero spec"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/demo.spec.ts", Line: 5, StartLine: 5, EndLine: 7, Name: `Outer > Inner > "focused spec"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/demo.spec.ts", Line: 10, StartLine: 10, EndLine: 10, Name: `Outer > Skipped > "skipped spec"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/demo.spec.ts", Line: 14, StartLine: 14, EndLine: 14, Name: `TplSuite > "tpl spec"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/edges.spec.ts", Line: 5, StartLine: 5, EndLine: 5, Name: `"after semi"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/edges.spec.ts", Line: 7, StartLine: 7, EndLine: 7, Name: `InlineSuite > "inline spec"`, Kind: "spec"},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/edges.spec.ts", Line: 9, StartLine: 8, EndLine: 9, Name: `"paren wrapped"`, Kind: "spec", Doc: "Paren-wrapped spec preserves comment."},
				{Target: "web/src/app/demo.spec.ts", File: "web/src/app/edges.spec.ts", Line: 13, StartLine: 10, EndLine: 13, Name: `"after block doc"`, Kind: "spec", Doc: "Block comment on spec."},
			},
		},
		// Skips untouched spec files and emits only specs whose comment-inclusive span overlaps changedLines.
		{
			name:     "differential mode extracts only overlapping specs",
			targetID: "web/src/app/touched.spec.ts",
			files: []BundledFile{
				{RelPath: "web/src/app/untouched.spec.ts", Content: []byte(diffSrc)},
				{RelPath: "web/src/app/touched.spec.ts", Content: []byte(diffSrc)},
			},
			changedLines: map[string]map[int]struct{}{
				"web/src/app/touched.spec.ts": {3: {}},
			},
			want: []ExtractedItem{
				{Target: "web/src/app/touched.spec.ts", File: "web/src/app/touched.spec.ts", Line: 4, StartLine: 3, EndLine: 4, Name: `"touched two"`, Kind: "spec", Doc: "Explains touched spec."},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractChangedTSSpecs(context.Background(), tc.targetID, webDir, tc.files, tc.changedLines)

			if err != nil {
				t.Fatalf("ExtractChangedTSSpecs() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractChangedTSSpecs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractChangedTSShortCircuit(t *testing.T) {
	testCases := []struct {
		name         string
		extract      func(context.Context, string, string, []BundledFile, map[string]map[int]struct{}) ([]ExtractedItem, error)
		files        []BundledFile
		changedLines map[string]map[int]struct{}
	}{
		// Returns nil without invoking node when no production TypeScript files match or overlap changedLines.
		{
			name:    "decls skips node when zero prod TS files match",
			extract: ExtractChangedTSDecls,
			files: []BundledFile{
				{RelPath: "web/src/app/readme.md", Content: []byte("# Doc")},
				{RelPath: "web/src/app/untouched.ts", Content: []byte("export function a() {}")},
			},
			changedLines: map[string]map[int]struct{}{"web/src/app/other.ts": {1: {}}},
		},
		// Returns nil without invoking node when no spec files match or overlap changedLines.
		{
			name:    "specs skips node when zero spec files match",
			extract: ExtractChangedTSSpecs,
			files: []BundledFile{
				{RelPath: "web/src/app/prod.ts", Content: []byte("it('ignored', () => {});")},
				{RelPath: "web/src/app/untouched.spec.ts", Content: []byte("it('untouched', () => {});")},
			},
			changedLines: map[string]map[int]struct{}{"web/src/app/other.spec.ts": {1: {}}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())

			got, err := tc.extract(context.Background(), "web/app", "/nonexistent/web", tc.files, tc.changedLines)

			if err != nil {
				t.Fatalf("extractor unexpected error: %v", err)
			}
			if diff := cmp.Diff([]ExtractedItem(nil), got); diff != "" {
				t.Errorf("extractor mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractChangedTSNodeFailures(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(t *testing.T) string
		wantErrSub string
	}{
		// Uses explicit webDir containing package.json and propagates node stderr when typescript cannot be resolved there.
		{
			name: "explicit webDir without typescript propagates node stderr",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"no-ts"}`), 0o600); err != nil {
					t.Fatalf("WriteFile(package.json) failed: %v", err)
				}
				return dir
			},
			wantErrSub: "Cannot find module",
		},
		// Returns a JSON decode error when node exits zero with malformed stdout.
		{
			name: "invalid JSON stdout from node",
			setup: func(t *testing.T) string {
				fakeBin := t.TempDir()
				fakeNode := filepath.Join(fakeBin, "node")
				script := `#!/bin/sh
printf '{bad-json'
`
				if err := os.WriteFile(fakeNode, []byte(script), 0o755); err != nil {
					t.Fatalf("WriteFile(fake node) failed: %v", err)
				}
				t.Setenv("PATH", fakeBin)
				return ""
			},
			wantErrSub: "failed decoding TypeScript AST output:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			webDir := tc.setup(t)
			files := []BundledFile{{RelPath: "web/src/app/a.ts", Content: []byte("export function f() {}")}}

			_, err := ExtractChangedTSDecls(context.Background(), "web/app", webDir, files, nil)

			if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("ExtractChangedTSDecls() error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

// TestExtractChangedTSCanceledContext verifies that TypeScript AST extraction fails immediately when ctx is already canceled.
func TestExtractChangedTSCanceledContext(t *testing.T) {
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name":"canceled"}`), 0o600); err != nil {
		t.Fatalf("WriteFile(package.json) failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	files := []BundledFile{{RelPath: "web/src/app/a.ts", Content: []byte("export function f() {}")}}

	_, err := ExtractChangedTSDecls(ctx, "web/app", webDir, files, nil)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ExtractChangedTSDecls(canceled) error = %v, want context.Canceled", err)
	}
}

func TestFormatTSExtractorError(t *testing.T) {
	baseErr := errors.New("exit status 1")
	testCases := []struct {
		name   string
		stderr string
		want   string
	}{
		// Includes trimmed stderr details when node writes an error message.
		{
			name:   "non-empty stderr included in error",
			stderr: "  syntax failure \n",
			want:   "typescript AST extractor failed (exit status 1): syntax failure",
		},
		// Omits trailing stderr details when stderr is empty or whitespace only.
		{
			name:   "whitespace-only stderr omitted",
			stderr: " \n\t ",
			want:   "typescript AST extractor failed: exit status 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTSExtractorError(baseErr, tc.stderr)

			if got.Error() != tc.want {
				t.Errorf("formatTSExtractorError() = %q, want %q", got.Error(), tc.want)
			}
		})
	}
}

func skipIfTypeScriptUnavailable(t *testing.T) string {
	t.Helper()
	webDir, err := gotestutil.ResolveWebDir("")
	if err != nil {
		t.Skipf("skipping TypeScript AST test: could not locate web directory (%v)", err)
	}
	if _, statErr := os.Stat(filepath.Join(webDir, "node_modules", "typescript")); statErr != nil {
		t.Skipf("skipping TypeScript AST test: web/node_modules/typescript not installed (%v)", statErr)
	}
	return webDir
}
