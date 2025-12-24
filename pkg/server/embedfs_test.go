// Copyright 2025 Google LLC
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

package server

import (
	"embed"
	"testing"
)

//go:embed dist/browser
var embeddedStaticFolderTest embed.FS

func TestEmbedFolder_Exists(t *testing.T) {
	testCases := []struct {
		desc   string
		prefix string
		path   string
		want   bool
	}{
		{
			desc:   "file exists at root",
			prefix: "/",
			path:   "/index.html",
			want:   true,
		},
		{
			desc:   "file exists at root with prefix",
			prefix: "/proxy/foo",
			path:   "/proxy/foo/index.html",
			want:   true,
		},
		{
			desc:   "file doesn't exists at root with prefix",
			prefix: "/proxy/foo",
			path:   "index.html",
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fs := embedFolder(embeddedStaticFolderTest, embeddedStaticFolderPath)

			got := fs.Exists(tc.prefix, tc.path)
			if got != tc.want {
				t.Errorf("Exists(%q, %q) = %v, want %v", tc.prefix, tc.path, got, tc.want)
			}
		})
	}
}
