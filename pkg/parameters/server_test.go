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

package parameters

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestServerParameters(t *testing.T) {
	testCases := []struct {
		name            string
		want            *ServerParameters
		wantErr         bool
		wantErrContains string
		before          func()
	}{
		{
			before: func() {
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "default",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath uses BasePath when not set",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/bar/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar", "--frontend-resource-base-path", "/foo"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath should complement the last /",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--max-loaded-inspections", "1"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "MaxLoadedInspections accepts 1",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(1),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--max-loaded-inspections", "0"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name:            "MaxLoadedInspections rejects values below 1",
			wantErr:         true,
			wantErrContains: "--max-loaded-inspections must be 1 or greater",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			store := &ServerParameters{}
			tc.before()
			ResetStore()
			AddStore(store)
			err := Parse()
			if tc.wantErr {
				if err == nil {
					t.Fatal("Parse() returned nil error, want an error")
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("Parse() error = %q, want error containing %q", err.Error(), tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, store); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}
