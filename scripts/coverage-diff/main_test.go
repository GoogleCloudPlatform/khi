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

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFlags(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		want    cliConfig
		wantErr bool
	}{
		{
			name: "defaults",
			args: nil,
			want: cliConfig{
				vcs:                "auto",
				format:             "text",
				maxAnnotations:     10,
				minAnnotationLines: 4,
			},
		},
		{
			name: "explicit flags",
			args: []string{"--vcs=git", "--base=HEAD^1", "--target=HEAD", "--format=github", "--min-annotation-lines=6"},
			want: cliConfig{
				vcs:                "git",
				base:               "HEAD^1",
				target:             "HEAD",
				format:             "github",
				maxAnnotations:     10,
				minAnnotationLines: 6,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFlags(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseFlags() error = %v, wantErr %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(cliConfig{})); diff != "" {
				t.Errorf("parseFlags() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
