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

package workbench

import (
	"testing"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// The following severity definitions emulate the style chunk of a loaded inspection in the tests of this package.
var (
	testSeverityInfo    = &khifilev6.Severity{Id: proto.Uint32(1), Label: proto.String("INFO"), Order: proto.Int32(1)}
	testSeverityWarning = &khifilev6.Severity{Id: proto.Uint32(2), Label: proto.String("WARNING"), Order: proto.Int32(2)}
	testSeverityError   = &khifilev6.Severity{Id: proto.Uint32(3), Label: proto.String("ERROR"), Order: proto.Int32(3)}
	testSeverityFatal   = &khifilev6.Severity{Id: proto.Uint32(4), Label: proto.String("FATAL"), Order: proto.Int32(4)}
)

func TestSeverityCounterSorted(t *testing.T) {
	customSeverityA := &khifilev6.Severity{Id: proto.Uint32(5), Label: proto.String("CUSTOM_A"), Order: proto.Int32(3)}
	customSeverityB := &khifilev6.Severity{Id: proto.Uint32(6), Label: proto.String("CUSTOM_B"), Order: proto.Int32(3)}

	testCases := []struct {
		name    string
		counter severityCounter
		want    []SeverityCount
	}{
		{
			name:    "empty counter returns nil",
			counter: severityCounter{},
			want:    nil,
		},
		{
			name: "counts are ordered from the most severe definition",
			counter: severityCounter{
				testSeverityInfo:    3,
				testSeverityFatal:   1,
				testSeverityWarning: 2,
				testSeverityError:   4,
			},
			want: []SeverityCount{
				{Severity: testSeverityFatal, Count: 1},
				{Severity: testSeverityError, Count: 4},
				{Severity: testSeverityWarning, Count: 2},
				{Severity: testSeverityInfo, Count: 3},
			},
		},
		{
			name: "definitions with the same order are ordered by ID",
			counter: severityCounter{
				customSeverityB: 1,
				customSeverityA: 2,
			},
			want: []SeverityCount{
				{Severity: customSeverityA, Count: 2},
				{Severity: customSeverityB, Count: 1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.counter.sorted()
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("sorted() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
