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

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
)

// TestServerParameters verifies default values, CLI flag parsing, environment variable fallbacks, and validation for ServerParameters.
func TestServerParameters(t *testing.T) {
	prevAllowedHosts, hadAllowedHosts := os.LookupEnv("KHI_ALLOWED_HOSTS")
	if err := os.Unsetenv("KHI_ALLOWED_HOSTS"); err != nil {
		t.Fatalf("os.Unsetenv(KHI_ALLOWED_HOSTS) failed: %v", err)
	}
	t.Cleanup(func() {
		if hadAllowedHosts {
			_ = os.Setenv("KHI_ALLOWED_HOSTS", prevAllowedHosts)
		} else {
			_ = os.Unsetenv("KHI_ALLOWED_HOSTS")
		}
	})

	testCases := []struct {
		name            string
		want            *ServerParameters
		wantErrContains string
		before          func(t *testing.T)
	}{
		// Uses default ServerParameters values when no flags or environment variables are set.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "default",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P(""),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Sets FrontendResourceBasePath from BasePath and appends a trailing slash when omitted.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath uses BasePath when not set",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P(""),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/bar/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Appends a trailing slash to an explicit FrontendResourceBasePath independently of BasePath.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--base-path", "/foo/bar", "--frontend-resource-base-path", "/foo"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "FrontendResourceBasePath should complement the last /",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P(""),
				BasePath:                 testutil.P("/foo/bar/"),
				FrontendResourceBasePath: testutil.P("/foo/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Accepts the minimum valid MaxLoadedInspections boundary value of 1.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--max-loaded-inspections", "1"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "MaxLoadedInspections accepts 1",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P(""),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(1),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Enables MCPDisableRenderer when --mcp-disable-renderer is passed on the CLI.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--mcp-disable-renderer"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "MCPDisableRenderer can be enabled via flag",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P(""),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(true),
			},
		},
		// Parses --allowed-hosts CLI flag into ServerParameters.AllowedHosts.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--allowed-hosts", "khi.example.com,*.proxy.example.com"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "AllowedHosts can be configured via flag",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P("khi.example.com,*.proxy.example.com"),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Reads KHI_ALLOWED_HOSTS into ServerParameters.AllowedHosts when the --allowed-hosts flag is omitted.
		{
			before: func(t *testing.T) {
				t.Setenv("KHI_ALLOWED_HOSTS", "app.example.com,*-dot-app.example.com")
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name: "AllowedHosts falls back to KHI_ALLOWED_HOSTS environment variable",
			want: &ServerParameters{
				Port:                     testutil.P(8080),
				Host:                     testutil.P("127.0.0.1"),
				AllowedHosts:             testutil.P("app.example.com,*-dot-app.example.com"),
				BasePath:                 testutil.P("/"),
				FrontendResourceBasePath: testutil.P("/"),
				FrontendAssetFolder:      testutil.P(""),
				MaxUploadFileSizeInBytes: testutil.P(1024 * 1024 * 1024),
				MaxLoadedInspections:     testutil.P(3),
				MCPDisableRenderer:       testutil.P(false),
			},
		},
		// Rejects MaxLoadedInspections values below 1 with a validation error during PostProcess.
		{
			before: func(t *testing.T) {
				os.Args = []string{os.Args[0], "--max-loaded-inspections", "0"}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
			name:            "MaxLoadedInspections rejects values below 1",
			wantErrContains: "--max-loaded-inspections must be 1 or greater",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			store := &ServerParameters{}
			tc.before(t)
			ResetStore()
			AddStore(store)
			err := Parse()
			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatal("Parse() returned nil error, want an error")
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
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
