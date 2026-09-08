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

package testchangeset

import (
	"testing"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
)

func TestTimelineChangeSetAsserter_HasEventCount(t *testing.T) {
	testCases := []struct {
		desc      string
		setupCS   func() *khifilev6.TimelineChangeSet
		wantCount int
	}{
		{
			desc: "zero events matches expected count of zero",
			setupCS: func() *khifilev6.TimelineChangeSet {
				return khifilev6.NewTimelineChangeSet(nil)
			},
			wantCount: 0,
		},
		{
			desc: "staged events count matches non-zero count",
			setupCS: func() *khifilev6.TimelineChangeSet {
				cs := khifilev6.NewTimelineChangeSet(nil)
				cs.AddEvent(&khifilev6.TimelinePath{ID: 1})
				cs.AddEvent(&khifilev6.TimelinePath{ID: 2})
				return cs
			},
			wantCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cs := tc.setupCS()
			AssertTimeline(t, cs).HasEventCount(tc.wantCount)
		})
	}
}
