// Copyright 2024 Google LLC
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

package queryutil

import (
	"fmt"
	"strings"
	"time"
)

func TimeRangeQuerySection(startTime time.Time, endTime time.Time, includeEnd bool) string {
	endEqual := ""
	if includeEnd {
		endEqual = "="
	}
	format := "2006-01-02T15:04:05-0700"
	return fmt.Sprintf(`timestamp >= "%s"
timestamp <%s "%s"`, startTime.Format(format), endEqual, endTime.Format(format))
}

func ToLowerForStringArray(source []string) []string {
	result := []string{}
	for _, s := range source {
		result = append(result, strings.ToLower(s))
	}
	return result
}

func WrapDoubleQuoteForStringArray(source []string) []string {
	result := []string{}
	for _, s := range source {
		result = append(result, fmt.Sprintf(`"%s"`, s))
	}
	return result
}

// SplitToChildGroups divices the given array with multiple child array not exceeding the given maxCount
func SplitToChildGroups[T any](input []T, maxCount int) [][]T {
	result := make([][]T, 0)
	for i, value := range input {
		if i/maxCount == len(result) {
			result = append(result, make([]T, 0))
		}
		result[i/maxCount] = append(result[i/maxCount], value)
	}
	return result
}
