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

package rule

const (
	// ViolationNone indicates the measured value is within budget or did not worsen an existing legacy value.
	ViolationNone ViolationKind = ""
	// ViolationHardLimit indicates a new or previously compliant unit exceeded the rule limit.
	ViolationHardLimit ViolationKind = "HARD_LIMIT"
	// ViolationRatchetRegression indicates an existing legacy unit worsened beyond its base revision value.
	ViolationRatchetRegression ViolationKind = "RATCHET_REGRESSION"
)

// ViolationKind classifies whether a rule sample passes, exceeds a hard limit, or worsens a legacy ratchet.
type ViolationKind string

// Sample holds the target value, configured limit, and optional base revision value for one rule check.
type Sample struct {
	// After is the measured metric value at the target revision.
	After int
	// Limit is the configured rule budget threshold.
	Limit int
	// Before is the measured metric value at the base revision.
	Before int
	// HasBefore reports whether Before is populated from the base revision.
	HasBefore bool
}

// ClassifyRatchet evaluates s under the unified KHI hard-limit and no-worsening ratchet model.
func ClassifyRatchet(s Sample) ViolationKind {
	if s.After <= s.Limit {
		return ViolationNone
	}
	if s.HasBefore && s.Before > s.Limit {
		if s.After > s.Before {
			return ViolationRatchetRegression
		}
		return ViolationNone
	}
	return ViolationHardLimit
}
