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

package parserutil

import "time"

// ParseRFC3339Time parses a string formatted in either RFC3339 or RFC3339Nano format
// and returns the parsed time. It first attempts to parse the input string using
// time.RFC3339Nano, and if that fails, it falls back to time.RFC3339.
func ParseRFC3339Time(input string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, input)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(time.RFC3339, input)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
