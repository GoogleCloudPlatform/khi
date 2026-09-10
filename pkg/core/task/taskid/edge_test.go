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

package taskid

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReferenceOption(t *testing.T) {
	testCases := []struct {
		name       string
		baseConfig DependencyConfig
		options    []ReferenceOption
		wantConfig DependencyConfig
	}{
		{
			name:       "default point-to-point config",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    nil,
			wantConfig: DependencyConfig{
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeAll,
			},
		},
		{
			name:       "point-to-point with ScopeActiveGraph",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveGraph,
			},
		},
		{
			name:       "point-to-point with ScopeActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
		},
		{
			name:       "point-to-point with ScopeAll (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeAll},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeAll,
			},
		},
		{
			name:       "point-to-point with duplicate ScopeActiveGraph (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveGraph, ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveGraph,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.baseConfig
			for _, opt := range tc.options {
				ApplyReferenceOption(&cfg, opt)
			}
			if diff := cmp.Diff(tc.wantConfig, cfg); diff != "" {
				t.Errorf("DependencyConfig mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFanInOption(t *testing.T) {
	testCases := []struct {
		name       string
		baseConfig DependencyConfig
		options    []FanInOption
		wantConfig DependencyConfig
	}{
		{
			name:       "default fan-in config",
			baseConfig: NewDefaultFanInConfig(),
			options:    nil,
			wantConfig: DependencyConfig{
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
		},
		{
			name:       "fan-in with ScopeActiveFeatures (idempotent)",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
		},
		{
			name:       "fan-in with ScopeActiveGraph",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveGraph,
			},
		},
		{
			name:       "fan-in with duplicate ScopeActiveGraph (idempotent)",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveGraph, ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveGraph,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.baseConfig
			for _, opt := range tc.options {
				ApplyFanInOption(&cfg, opt)
			}
			if diff := cmp.Diff(tc.wantConfig, cfg); diff != "" {
				t.Errorf("DependencyConfig mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFanInOptionPanic(t *testing.T) {
	testCases := []struct {
		name       string
		baseConfig DependencyConfig
		options    []FanInOption
	}{
		{
			name:       "ScopeAll is not supported for fan-in",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeAll},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected panic for %s, but did not panic", tc.name)
				}
			}()

			cfg := tc.baseConfig
			for _, opt := range tc.options {
				ApplyFanInOption(&cfg, opt)
			}
		})
	}
}
