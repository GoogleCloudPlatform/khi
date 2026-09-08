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
		name              string
		baseConfig        DependencyConfig
		options           []ReferenceOption
		wantConfig        DependencyConfig
		wantResolvedScope DependencyScope
	}{
		{
			name:       "default point-to-point config",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    nil,
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "point-to-point with Optional",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "point-to-point with duplicate Optional (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, Optional},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "point-to-point with ScopeActiveGraph",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveGraph,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "point-to-point with Optional and ScopeActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with ScopeActiveFeatures and Optional",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures, Optional},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with Optional and ScopeAll",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, ScopeAll},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeAll,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "point-to-point with ScopeActiveFeatures only",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with duplicate ScopeActiveFeatures (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures, ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with OrderOnly",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{OrderOnly},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindOrderOnly,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "point-to-point with duplicate OrderOnly (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{OrderOnly, OrderOnly},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindOrderOnly,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "point-to-point with Optional, ScopeActiveFeatures, and OrderOnly",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, ScopeActiveFeatures, OrderOnly},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindOrderOnly,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
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
			if got := cfg.ResolvedScope(); got != tc.wantResolvedScope {
				t.Errorf("ResolvedScope() = %v, want %v", got, tc.wantResolvedScope)
			}
		})
	}
}

func TestFanInOption(t *testing.T) {
	testCases := []struct {
		name              string
		baseConfig        DependencyConfig
		options           []FanInOption
		wantConfig        DependencyConfig
		wantResolvedScope DependencyScope
	}{
		{
			name:       "default fan-in config",
			baseConfig: NewDefaultFanInConfig(),
			options:    nil,
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "fan-in with ScopeActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "fan-in with duplicate ScopeActiveFeatures (idempotent)",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveFeatures, ScopeActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "fan-in with ScopeAll and OrderOnly",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeAll, OrderOnly},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindOrderOnly,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeAll,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "fan-in with duplicate OrderOnly (idempotent)",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{OrderOnly, OrderOnly},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindOrderOnly,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeUnspecified,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "fan-in with ScopeActiveGraph",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveGraph},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveGraph,
			},
			wantResolvedScope: ScopeActiveGraph,
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
			if got := cfg.ResolvedScope(); got != tc.wantResolvedScope {
				t.Errorf("ResolvedScope() = %v, want %v", got, tc.wantResolvedScope)
			}
		})
	}
}

func TestReferenceOptionPanic(t *testing.T) {
	testCases := []struct {
		name       string
		baseConfig DependencyConfig
		options    []ReferenceOption
	}{
		{
			name:       "ScopeAll called after ScopeActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures, ScopeAll},
		},
		{
			name:       "ScopeActiveFeatures called after ScopeAll",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeAll, ScopeActiveFeatures},
		},
		{
			name:       "ScopeActiveGraph called after ScopeActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{ScopeActiveFeatures, ScopeActiveGraph},
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
				ApplyReferenceOption(&cfg, opt)
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
			name:       "ScopeAll called after ScopeActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveFeatures, ScopeAll},
		},
		{
			name:       "ScopeActiveFeatures called after ScopeAll",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeAll, ScopeActiveFeatures},
		},
		{
			name:       "ScopeActiveGraph called after ScopeActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{ScopeActiveFeatures, ScopeActiveGraph},
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
