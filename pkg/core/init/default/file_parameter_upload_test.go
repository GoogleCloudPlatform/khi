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

package defaultinit

import "testing"

func TestUploadURLBase(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		port     int
		basePath string
		want     string
	}{
		{
			name: "loopback IPv4 host is used as is",
			host: "127.0.0.1",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv4 wildcard host is replaced with loopback",
			host: "0.0.0.0",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv6 wildcard host is replaced with loopback",
			host: "::",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "empty host is replaced with loopback",
			host: "",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv6 host is bracketed",
			host: "::1",
			port: 8080,
			want: "http://[::1]:8080/api/v1/file-upload/",
		},
		{
			name:     "base path is placed before the upload path",
			host:     "localhost",
			port:     9000,
			basePath: "/khi",
			want:     "http://localhost:9000/khi/api/v1/file-upload/",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := uploadURLBase(tc.host, tc.port, tc.basePath)
			if got != tc.want {
				t.Errorf("uploadURLBase(%q, %d, %q) = %q, want %q", tc.host, tc.port, tc.basePath, got, tc.want)
			}
		})
	}
}
