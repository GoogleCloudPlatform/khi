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
	"cmp"
	"slices"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// SeverityCount represents the number of logs with a severity defined in the style chunk of the inspection.
type SeverityCount struct {
	Severity *khifilev6.Severity
	Count    int
}

// severityCounter counts logs per severity definition in the style chunk of the inspection.
type severityCounter map[*khifilev6.Severity]int

// sorted returns the counts in descending severity order, breaking ties by severity ID so that the output is deterministic.
func (c severityCounter) sorted() []SeverityCount {
	var counts []SeverityCount
	for severity, count := range c {
		counts = append(counts, SeverityCount{Severity: severity, Count: count})
	}
	slices.SortFunc(counts, func(a, b SeverityCount) int {
		return cmp.Or(
			cmp.Compare(b.Severity.GetOrder(), a.Severity.GetOrder()),
			cmp.Compare(a.Severity.GetId(), b.Severity.GetId()),
		)
	})
	return counts
}
