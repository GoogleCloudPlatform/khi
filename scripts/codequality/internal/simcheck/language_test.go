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

package simcheck

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type stubRegistryAnalyzer struct {
	id    LanguageID
	title string
}

func (s stubRegistryAnalyzer) ID() LanguageID                               { return s.id }
func (s stubRegistryAnalyzer) DisplayName() string                          { return s.title }
func (s stubRegistryAnalyzer) CodeFence() string                            { return string(s.id) }
func (s stubRegistryAnalyzer) IsEligibleFile(string) bool                   { return false }
func (s stubRegistryAnalyzer) DiscoverCorpusFiles(string) ([]string, error) { return nil, nil }
func (s stubRegistryAnalyzer) IsSharedModule(string) bool                   { return false }
func (s stubRegistryAnalyzer) ExtractUnits(context.Context, []FileInput, int, int) (map[string][]Unit, error) {
	return nil, nil
}
func (s stubRegistryAnalyzer) ExtractBaseKeys(context.Context, []FileInput) (map[string]map[string]struct{}, error) {
	return nil, nil
}

func TestRegistryResolveAndReplace(t *testing.T) {
	reg := NewRegistry(
		nil,
		stubRegistryAnalyzer{id: LangGo, title: "GoOld"},
		stubRegistryAnalyzer{id: LangTS, title: "TypeScript"},
		stubRegistryAnalyzer{id: LangGo, title: "GoUpdated"},
	)

	testCases := []struct {
		name       string
		flag       string
		wantTitles []string
		wantErr    bool
	}{
		{
			name:       "empty flag resolves all in registration order with updated instance",
			flag:       "",
			wantTitles: []string{"GoUpdated", "TypeScript"},
		},
		{
			name:       "all keyword resolves all registered languages",
			flag:       " all ",
			wantTitles: []string{"GoUpdated", "TypeScript"},
		},
		{
			name:       "case-insensitive single language lookup",
			flag:       "TS",
			wantTitles: []string{"TypeScript"},
		},
		{
			name:    "unknown language returns descriptive error",
			flag:    "rust",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotAnalyzers, err := reg.Resolve(tc.flag)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Resolve(%q) err = %v, wantErr %v", tc.flag, err, tc.wantErr)
			}
			if tc.wantErr && err.Error() != `invalid --lang "rust" (want all, go, ts)` {
				t.Errorf("Resolve(%q) err = %q, want exact allowed list", tc.flag, err.Error())
			}
			var gotTitles []string
			for _, a := range gotAnalyzers {
				gotTitles = append(gotTitles, a.DisplayName())
			}
			if diff := cmp.Diff(tc.wantTitles, gotTitles); diff != "" {
				t.Errorf("Resolve(%q) titles mismatch (-want +got):\n%s", tc.flag, diff)
			}
		})
	}
}
