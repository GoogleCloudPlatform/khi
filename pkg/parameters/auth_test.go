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
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestAuthParameters(t *testing.T) {
	// Test environment gives the token from environment variable and it can affect this test case.
	// The value will be kept in the variable and restored after testing them.
	accessTokenBefore := os.Getenv("GCP_ACCESS_TOKEN")
	os.Setenv("GCP_ACCESS_TOKEN", "")
	defer os.Setenv("GCP_ACCESS_TOKEN", accessTokenBefore)

	testCases := []struct {
		name   string
		want   *AuthParameters
		before func()
	}{
		{
			name: "default",
			want: &AuthParameters{
				AccessToken:                    testutil.P(""),
				DisableMetadataServer:          testutil.P(false),
				FixedProjectID:                 testutil.P(""),
				QuotaProjectID:                 testutil.P(""),
				OAuthClientID:                  testutil.P(""),
				OAuthClientSecret:              testutil.P(""),
				OAuthRedirectURI:               testutil.P(""),
				OAuthRedirectTargetServingPath: testutil.P("/oauth/callback"),
				OAuthStateSuffix:               testutil.P(""),
			},
			before: func() {
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			tc.before()
			store := &AuthParameters{}
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

func TestAuthParameters_GetOAuthConfig(t *testing.T) {
	testCases := []struct {
		name     string
		params   *AuthParameters
		expected *oauth2.Config
	}{
		{
			name: "with values",
			params: &AuthParameters{
				OAuthClientID:     testutil.P("client-id"),
				OAuthClientSecret: testutil.P("client-secret"),
				OAuthRedirectURI:  testutil.P("https://example.com/callback"),
			},
			expected: &oauth2.Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://example.com/callback",
				Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform"},
				Endpoint:     google.Endpoint,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			got := tc.params.GetOAuthConfig()
			if diff := cmp.Diff(tc.expected, got, cmpopts.IgnoreUnexported(oauth2.Config{})); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}

}

func TestAuthParameters_OAuthEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		params   *AuthParameters
		expected bool
	}{
		{
			name: "all set",
			params: &AuthParameters{
				OAuthClientID:                  testutil.P("client-id"),
				OAuthClientSecret:              testutil.P("client-secret"),
				OAuthRedirectURI:               testutil.P("https://example.com/callback"),
				OAuthRedirectTargetServingPath: testutil.P("/oauth/callback"),
			},
			expected: true,
		},
		{
			name: "client id is missing",
			params: &AuthParameters{
				OAuthClientID:                  testutil.P(""),
				OAuthClientSecret:              testutil.P("client-secret"),
				OAuthRedirectURI:               testutil.P("https://example.com/callback"),
				OAuthRedirectTargetServingPath: testutil.P("/oauth/callback"),
			},
			expected: false,
		},
		{
			name: "client secret is missing",
			params: &AuthParameters{
				OAuthClientID:                  testutil.P("client-id"),
				OAuthClientSecret:              testutil.P(""),
				OAuthRedirectURI:               testutil.P("https://example.com/callback"),
				OAuthRedirectTargetServingPath: testutil.P("/oauth/callback"),
			},
			expected: false,
		},
		{
			name: "callback url is missing",
			params: &AuthParameters{
				OAuthClientID:                  testutil.P("client-id"),
				OAuthClientSecret:              testutil.P("client-secret"),
				OAuthRedirectURI:               testutil.P(""),
				OAuthRedirectTargetServingPath: testutil.P("/oauth/callback"),
			},
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			got := tc.params.OAuthEnabled()
			if got != tc.expected {
				t.Errorf("unexpected result, got %v, want %v", got, tc.expected)
			}
		})
	}
}
