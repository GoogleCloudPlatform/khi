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
	"fmt"
	"sort"
)

// ByteRange represents an uploaded byte interval [Start, End).
type ByteRange struct {
	Start int64
	End   int64
}

// ValidateReceivedRanges validates that the given byte ranges completely cover [0, expectedTotalSize)
// without gaps or overlaps after sorting by start offset.
func ValidateReceivedRanges(ranges []ByteRange, expectedTotalSize int64) error {
	if len(ranges) == 0 {
		return fmt.Errorf("incomplete upload: no data received")
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	if ranges[0].Start != 0 {
		return fmt.Errorf("incomplete upload: first chunk starts at offset %d, expected 0", ranges[0].Start)
	}

	for i := 0; i < len(ranges)-1; i++ {
		if ranges[i].End != ranges[i+1].Start {
			return fmt.Errorf("incomplete or overlapping upload: chunk %d ends at %d but next chunk starts at %d",
				i, ranges[i].End, ranges[i+1].Start)
		}
	}

	lastEnd := ranges[len(ranges)-1].End
	if lastEnd != expectedTotalSize {
		return fmt.Errorf("incomplete upload: received up to byte %d, expected %d", lastEnd, expectedTotalSize)
	}

	return nil
}
