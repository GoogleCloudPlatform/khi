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

package k8saudit

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

func TestResolveResourceName(t *testing.T) {
	testCases := []struct {
		desc              string
		nameFromLogFields string
		requestYAML       string
		responseYAML      string
		want              string
	}{
		{
			desc:              "nameFromLogFields is already present",
			nameFromLogFields: "explicit-pod",
			requestYAML: `metadata:
  name: request-pod
  generateName: gen-pod-`,
			responseYAML: `metadata:
  name: response-pod`,
			want: "explicit-pod",
		},
		{
			desc:              "falls back to response metadata.name when nameFromLogFields is empty",
			nameFromLogFields: "",
			requestYAML: `metadata:
  generateName: nginx-deployment-abcdef-`,
			responseYAML: `apiVersion: v1
kind: Pod
metadata:
  name: nginx-deployment-abcdef-12345`,
			want: "nginx-deployment-abcdef-12345",
		},
		{
			desc:              "ignores Status response and falls back to request metadata.name",
			nameFromLogFields: "",
			requestYAML: `metadata:
  name: nginx-pod`,
			responseYAML: `apiVersion: v1
kind: Status
metadata:
  name: ignored-status-name`,
			want: "nginx-pod",
		},
		{
			desc:              "falls back to request metadata.generateName with wildcard suffix when nameFromLogFields and request name are empty",
			nameFromLogFields: "",
			requestYAML: `metadata:
  generateName: nginx-deployment-abcdef-`,
			responseYAML: `apiVersion: v1
kind: Status
status: Failure`,
			want: "nginx-deployment-abcdef-?????",
		},
		{
			desc:              "returns empty string when no name or generateName is available",
			nameFromLogFields: "",
			requestYAML:       `{}`,
			responseYAML:      `{}`,
			want:              "",
		},
		{
			desc:              "returns empty string when request and response readers are nil",
			nameFromLogFields: "",
			requestYAML:       "",
			responseYAML:      "",
			want:              "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var reqReader, respReader *structured.NodeReader
			if tc.requestYAML != "" {
				node, err := structured.FromYAML(tc.requestYAML)
				if err != nil {
					t.Fatalf("failed to parse requestYAML: %v", err)
				}
				reqReader = structured.NewNodeReader(node)
			}
			if tc.responseYAML != "" {
				node, err := structured.FromYAML(tc.responseYAML)
				if err != nil {
					t.Fatalf("failed to parse responseYAML: %v", err)
				}
				respReader = structured.NewNodeReader(node)
			}

			got := ResolveResourceName(tc.nameFromLogFields, reqReader, respReader)
			if got != tc.want {
				t.Errorf("ResolveResourceName() = %q, want %q", got, tc.want)
			}
		})
	}
}
