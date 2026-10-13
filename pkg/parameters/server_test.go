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
				PublicURL:                testutil.P(""),
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
				PublicURL:                testutil.P(""),
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
				PublicURL:                testutil.P(""),
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
				PublicURL:                testutil.P(""),
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
			t.Setenv("KHI_PUBLIC_URL", "")
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

func TestServerParameters_PublicURL(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		env             string
		want            string
		wantErrContains string
	}{
		{name: "unset"},
		{name: "HTTP URL", args: []string{"--public-url=http://khi.example.com:9000"}, want: "http://khi.example.com:9000"},
		{name: "HTTPS URL with prefix", args: []string{"--public-url=https://khi.example.com/proxy/khi/"}, want: "https://khi.example.com/proxy/khi/"},
		{name: "IPv6 URL", args: []string{"--public-url=http://[::1]:8080/khi"}, want: "http://[::1]:8080/khi"},
		{name: "escaped path", args: []string{"--public-url=https://khi.example.com/team%20one"}, want: "https://khi.example.com/team%20one"},
		{name: "environment variable", env: "https://khi.example.com/khi", want: "https://khi.example.com/khi"},
		{name: "flag overrides environment", args: []string{"--public-url=http://localhost:9000"}, env: "https://khi.example.com/khi", want: "http://localhost:9000"},
		{name: "explicit empty flag overrides environment", args: []string{"--public-url="}, env: "https://khi.example.com/khi"},
		{name: "relative URL", args: []string{"--public-url=/khi"}, wantErrContains: "--public-url must be an absolute HTTP or HTTPS URL"},
		{name: "unsupported scheme", args: []string{"--public-url=ftp://khi.example.com"}, wantErrContains: "--public-url must be an absolute HTTP or HTTPS URL"},
		{name: "missing host", args: []string{"--public-url=https:///khi"}, wantErrContains: "--public-url must be an absolute HTTP or HTTPS URL"},
		{name: "empty hostname with port", args: []string{"--public-url=http://:8080"}, wantErrContains: "--public-url must be an absolute HTTP or HTTPS URL"},
		{name: "credentials", args: []string{"--public-url=https://user:password@khi.example.com"}, wantErrContains: "without credentials, a query, or a fragment"},
		{name: "query", args: []string{"--public-url=https://khi.example.com/khi?key=value"}, wantErrContains: "without credentials, a query, or a fragment"},
		{name: "empty query", args: []string{"--public-url=https://khi.example.com/khi?"}, wantErrContains: "without credentials, a query, or a fragment"},
		{name: "fragment", args: []string{"--public-url=https://khi.example.com/khi#section"}, wantErrContains: "without credentials, a query, or a fragment"},
		{name: "empty fragment", args: []string{"--public-url=https://khi.example.com/khi#"}, wantErrContains: "without credentials, a query, or a fragment"},
		{name: "invalid escape", args: []string{"--public-url=https://khi.example.com/%zz"}, wantErrContains: "--public-url is invalid"},
		{name: "invalid environment URL", env: "khi.example.com", wantErrContains: "--public-url must be an absolute HTTP or HTTPS URL"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			t.Setenv("KHI_PUBLIC_URL", tc.env)
			originalArgs, originalCommandLine := os.Args, flag.CommandLine
			t.Cleanup(func() {
				os.Args, flag.CommandLine = originalArgs, originalCommandLine
			})
			os.Args = append([]string{originalArgs[0]}, tc.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			store := &ServerParameters{}
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
				t.Fatalf("Parse() failed: %v", err)
			}
			if *store.PublicURL != tc.want {
				t.Errorf("PublicURL = %q, want %q", *store.PublicURL, tc.want)
			}
		})
	}
}
