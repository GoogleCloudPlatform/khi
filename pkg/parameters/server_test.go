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

package parameters

import (
	"flag"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestServerParameters(t *testing.T) {
	testCases := []struct {
		name   string
		want   *ServerParameters
		before func()
	}{
		{
			before: func() {
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "default",
			want: &ServerParameters{
				ViewerMode:               testutil.P(false),
				Port:                     testutil.P(8080),
				Host:                     testutil.P("localhost"),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P("./web"),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath uses BasePath when not set",
			want: &ServerParameters{
				ViewerMode:               testutil.P(false),
				Port:                     testutil.P(8080),
				Host:                     testutil.P("localhost"),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/bar/"),
				FrontendAssetFolder:      testutil.P("./web"),
			},
		},
		{
			before: func() {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar", "--frontend-resource-base-path", "/foo"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath should complement the last /",
			want: &ServerParameters{
				ViewerMode:               testutil.P(false),
				Port:                     testutil.P(8080),
				Host:                     testutil.P("localhost"),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/"),
				FrontendAssetFolder:      testutil.P("./web"),
			},
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
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, store); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}
