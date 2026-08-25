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

package chunkedupload

import (
	"testing"
)

func TestValidateReceivedRanges(t *testing.T) {
	testCases := []struct {
		name              string
		ranges            []ByteRange
		expectedTotalSize int64
		wantErr           bool
	}{
		{
			name: "valid sequential chunks",
			ranges: []ByteRange{
				{Start: 0, End: 5},
				{Start: 5, End: 10},
			},
			expectedTotalSize: 10,
			wantErr:           false,
		},
		{
			name: "valid out-of-order chunks",
			ranges: []ByteRange{
				{Start: 5, End: 10},
				{Start: 0, End: 5},
			},
			expectedTotalSize: 10,
			wantErr:           false,
		},
		{
			name:              "empty ranges",
			ranges:            []ByteRange{},
			expectedTotalSize: 10,
			wantErr:           true,
		},
		{
			name: "first chunk not starting at 0",
			ranges: []ByteRange{
				{Start: 2, End: 10},
			},
			expectedTotalSize: 10,
			wantErr:           true,
		},
		{
			name: "gap between chunks",
			ranges: []ByteRange{
				{Start: 0, End: 4},
				{Start: 6, End: 10},
			},
			expectedTotalSize: 10,
			wantErr:           true,
		},
		{
			name: "overlapping chunks",
			ranges: []ByteRange{
				{Start: 0, End: 6},
				{Start: 5, End: 10},
			},
			expectedTotalSize: 10,
			wantErr:           true,
		},
		{
			name: "last chunk does not reach total size",
			ranges: []ByteRange{
				{Start: 0, End: 5},
				{Start: 5, End: 8},
			},
			expectedTotalSize: 10,
			wantErr:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReceivedRanges(tc.ranges, tc.expectedTotalSize)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateReceivedRanges() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
