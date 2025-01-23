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

package testutil

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

/**
* KHI has many tests using configurations and these configurations are read by glob.
* glob result can contain non meaningful relative path and slash.
 */
func GlobResultToChildTestName(filePath string, basePath string) string {
	result := strings.TrimPrefix(filePath, basePath)
	result = strings.ReplaceAll(result, "/", "_")
	return result
}

// Removes timestamps from slog logs
func RemoveSlogTimestampFromLine(log string) string {
	logs := strings.Split(log, "\n")
	for i := 0; i < len(logs); i++ {
		if logs[i] != "" {
			if logs[i][28] == 'Z' {
				logs[i] = logs[i][30:] // the first 29 chars are used for timestamps for logs with Zt
				continue
			}
			logs[i] = logs[i][35:] // the first 35 chars are used for timestamps for logs with time offset
		}
	}
	return strings.Join(logs, "\n")
}

func ResponseFromString(code int, response string) *http.Response {
	return &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(response)),
		StatusCode: code,
	}
}

// P returns the pointer of given value. This is helper to get the pointer from literal because Go doesn't allow &"foo".
func P[T any](value T) *T {
	return &value
}
