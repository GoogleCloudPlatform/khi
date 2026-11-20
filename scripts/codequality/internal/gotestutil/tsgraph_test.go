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

package gotestutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlanTSTestExecution(t *testing.T) {
	tmpRepo := t.TempDir()

	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "main.ts"), `import { routes } from './app/app.route';
export const app = routes;
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "main.spec.ts"), `import { app } from './main';
describe('main', () => { it('works', () => expect(app).toBeDefined()); });
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "app.route.ts"), `import { FeatureSmartComponent } from 'src/app/feature/feature-smart.component';
export const routes = [FeatureSmartComponent];
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "feature", "types", "feature.viewmodel.ts"), `export function buildViewModel(v: number): string {
  return String(v);
}
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "feature", "feature-smart.component.ts"), `import { buildViewModel } from 'src/app/feature/types/feature.viewmodel';
export class FeatureSmartComponent {
  readonly vm = buildViewModel(1);
}
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "feature", "feature-smart.component.spec.ts"), `import { FeatureSmartComponent } from './feature-smart.component';
describe('FeatureSmartComponent', () => { it('creates', () => expect(new FeatureSmartComponent()).toBeTruthy()); });
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "barrel", "index.ts"), `import { helperVal } from './cycle-a.js';
export const barrelVal = helperVal;
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "barrel", "cycle-a.ts"), `import { bVal } from './cycle-b';
export const helperVal = bVal;
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "barrel", "cycle-b.ts"), `import { helperVal } from './cycle-a';
export const bVal = () => helperVal;
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "barrel", "consumer.spec.ts"), `import { barrelVal } from '../barrel';
describe('barrel', () => { it('loads', () => expect(barrelVal).toBeDefined()); });
`)
	mustWriteTSFixture(t, filepath.Join(tmpRepo, "web", "src", "app", "utils", "orphan.ts"), `export function orphan(v: number): number {
  return v + 1;
}
`)

	testCases := []struct {
		name         string
		changedFiles []string
		full         bool
		want         TSTestPlan
	}{
		{
			name:         "no changed files skips karma in targeted mode",
			changedFiles: nil,
			full:         false,
			want:         TSTestPlan{SkipKarma: true},
		},
		{
			name:         "full mode runs entire karma suite without include filter",
			changedFiles: nil,
			full:         true,
			want:         TSTestPlan{SkipKarma: false},
		},
		{
			name:         "transitive reverse import resolves feature spec while excluding main.spec.ts",
			changedFiles: []string{"web/src/app/feature/types/feature.viewmodel.ts"},
			full:         false,
			want: TSTestPlan{
				SkipKarma:    false,
				IncludeSpecs: []string{"src/app/feature/feature-smart.component.spec.ts"},
			},
		},
		{
			name:         "direct change to app.route.ts traverses main.ts to include main.spec.ts",
			changedFiles: []string{"web/src/app/app.route.ts"},
			full:         false,
			want: TSTestPlan{
				SkipKarma:    false,
				IncludeSpecs: []string{"src/main.spec.ts"},
			},
		},
		{
			name:         "cyclic barrel imports resolve without infinite recursion",
			changedFiles: []string{"web/src/app/barrel/cycle-b.ts"},
			full:         false,
			want: TSTestPlan{
				SkipKarma:    false,
				IncludeSpecs: []string{"src/app/barrel/consumer.spec.ts"},
			},
		},
		{
			name:         "unimported file skips karma launch",
			changedFiles: []string{"web/src/app/utils/orphan.ts"},
			full:         false,
			want:         TSTestPlan{SkipKarma: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PlanTSTestExecution(tmpRepo, tc.changedFiles, tc.full)
			if err != nil {
				t.Fatalf("PlanTSTestExecution() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("PlanTSTestExecution() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustWriteTSFixture(t *testing.T, absPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", absPath, err)
	}
}
