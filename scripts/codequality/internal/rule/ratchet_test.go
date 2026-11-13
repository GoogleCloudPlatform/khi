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

import "testing"

func TestClassifyRatchet(t *testing.T) {
	testCases := []struct {
		name   string
		sample Sample
		want   ViolationKind
	}{
		{
			name:   "new unit at exact limit passes",
			sample: Sample{After: 15, Limit: 15},
			want:   ViolationNone,
		},
		{
			name:   "new unit exceeding limit fails hard limit",
			sample: Sample{After: 16, Limit: 15},
			want:   ViolationHardLimit,
		},
		{
			name:   "existing compliant unit at exact limit crossing fails hard limit",
			sample: Sample{After: 16, Limit: 15, Before: 15, HasBefore: true},
			want:   ViolationHardLimit,
		},
		{
			name:   "legacy unit above limit unchanged passes ratchet",
			sample: Sample{After: 20, Limit: 15, Before: 20, HasBefore: true},
			want:   ViolationNone,
		},
		{
			name:   "legacy unit above limit worsened fails ratchet regression",
			sample: Sample{After: 21, Limit: 15, Before: 20, HasBefore: true},
			want:   ViolationRatchetRegression,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRatchet(tc.sample)
			if got != tc.want {
				t.Errorf("ClassifyRatchet(%+v) = %q, want %q", tc.sample, got, tc.want)
			}
		})
	}
}
