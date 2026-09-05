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
			name:       "point-to-point with FromActiveGraph",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveGraph},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveGraph,
			},
			wantResolvedScope: ScopeActiveGraph,
		},
		{
			name:       "point-to-point with Optional and FromActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, FromActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with FromActiveFeatures and Optional",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveFeatures, Optional},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with Optional and FromAll",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, FromAll},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionOptional,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeAll,
			},
			wantResolvedScope: ScopeAll,
		},
		{
			name:       "point-to-point with FromActiveFeatures only",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityPointToPoint,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "point-to-point with duplicate FromActiveFeatures (idempotent)",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveFeatures, FromActiveFeatures},
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
			name:       "point-to-point with Optional, FromActiveFeatures, and OrderOnly",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{Optional, FromActiveFeatures, OrderOnly},
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
			if diff := cmp.Diff(tc.wantResolvedScope, cfg.ResolvedScope()); diff != "" {
				t.Errorf("ResolvedScope() mismatch (-want +got):\n%s", diff)
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
			name:       "fan-in with FromActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "fan-in with duplicate FromActiveFeatures (idempotent)",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromActiveFeatures, FromActiveFeatures},
			wantConfig: DependencyConfig{
				Kind:        EdgeKindData,
				Condition:   ConditionRequired,
				Cardinality: CardinalityFanIn,
				Scope:       ScopeActiveFeatures,
			},
			wantResolvedScope: ScopeActiveFeatures,
		},
		{
			name:       "fan-in with FromAll and OrderOnly",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromAll, OrderOnly},
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
			name:       "fan-in with FromActiveGraph",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromActiveGraph},
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
			if diff := cmp.Diff(tc.wantResolvedScope, cfg.ResolvedScope()); diff != "" {
				t.Errorf("ResolvedScope() mismatch (-want +got):\n%s", diff)
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
			name:       "FromAll called after FromActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveFeatures, FromAll},
		},
		{
			name:       "FromActiveFeatures called after FromAll",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromAll, FromActiveFeatures},
		},
		{
			name:       "FromActiveGraph called after FromActiveFeatures",
			baseConfig: NewDefaultPointToPointConfig(),
			options:    []ReferenceOption{FromActiveFeatures, FromActiveGraph},
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
			name:       "FromAll called after FromActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromActiveFeatures, FromAll},
		},
		{
			name:       "FromActiveFeatures called after FromAll",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromAll, FromActiveFeatures},
		},
		{
			name:       "FromActiveGraph called after FromActiveFeatures",
			baseConfig: NewDefaultFanInConfig(),
			options:    []FanInOption{FromActiveFeatures, FromActiveGraph},
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
