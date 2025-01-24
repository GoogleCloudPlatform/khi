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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var Auth *AuthParameters = &AuthParameters{}

type AuthParameters struct {
	// AccessToken is the token used for GCP related requests.
	AccessToken *string

	// DisableMetadataServer
	// If this flag is set, KHI won't try to get access token from metadata server.
	DisableMetadataServer *bool

	// FixedProjectID is a GCP project ID prefilled in the form. User won't be able to edit it from the form.
	FixedProjectID *string

	// QuotaProjectID is a GCP project ID used as the quota project. This is useful when user wants to use KHI against a project with another project with larger logging read quota.
	QuotaProjectID *string

	// OAuthClientID is the client ID used for getting access tokens via OAuth.
	OAuthClientID *string

	// OAuthClientSecret is the client secret used for getting access tokens via OAuth.
	OAuthClientSecret *string

	// OAuthRedirectURI is the callback URL for OAuth. This must be provided as full qualified URL.
	OAuthRedirectURI *string

	// OAuthRedirectTargetServingPath is the path to serve the callback target.
	OAuthRedirectTargetServingPath *string

	// OAuthStateSuffix is the suffix added to the state parameter in OAuth. The state will be generated in the format of `<random-string><suffix>`.
	OAuthStateSuffix *string
}

// PostProcess implements ParameterStore.
func (a *AuthParameters) PostProcess() error {
	if *a.OAuthClientID != "" && *a.OAuthClientSecret == "" {
		return fmt.Errorf("--oauth-client-secret must be set when --oauth-client-id is set")
	}
	if *a.OAuthClientID == "" && *a.OAuthClientSecret != "" {
		return fmt.Errorf("--oauth-client-id must be set when --oauth-client-secret is set")
	}
	if *a.OAuthClientID != "" && *a.OAuthRedirectURI == "" {
		return fmt.Errorf("--oauth-redirect-uri must be set when --oauth-client-id is set")
	}
	return nil
}

// Prepare implements ParameterStore.
func (a *AuthParameters) Prepare() error {
	a.AccessToken = flag.String("access-token", "", "The token used for GCP related requests.", "GCP_ACCESS_TOKEN")
	a.DisableMetadataServer = flag.Bool("disable-metadata-server", false, "If this flag is set, KHI won't try to get access token from metadata server.", "")
	a.FixedProjectID = flag.String("fixed-project-id", "", "A GCP project ID prefilled in the form. User won't be able to edit it from the form.", "KHI_FIXED_PROJECT_ID")
	a.QuotaProjectID = flag.String("quota-project-id", "", "A GCP project ID used as the quota project. This is useful when user wants to use KHI against a project with another project with larger logging read quota.", "")
	a.OAuthClientID = flag.String("oauth-client-id", "", "The client ID used for getting access tokens via OAuth.", "KHI_OAUTH_CLIENT_ID")
	a.OAuthClientSecret = flag.String("oauth-client-secret", "", "The client secret used for getting access tokens via OAuth.", "KHI_OAUTH_CLIENT_SECRET")
	a.OAuthRedirectURI = flag.String("oauth-redirect-uri", "", "The callback URI for OAuth. This must be provided as full qualified URL.", "")
	a.OAuthRedirectTargetServingPath = flag.String("oauth-redirect-target-serving-path", "/oauth/callback", "The path to serve the callback target.", "")
	a.OAuthStateSuffix = flag.String("oauth-state-suffix", "", "The suffix added to the state parameter in OAuth. The state will be generated in the format of `<random-string><suffix>`.", "")
	return nil
}

// OAuthEnabled returns if the oauth based access token resolution is enabled or not.
func (a *AuthParameters) OAuthEnabled() bool {
	if a.OAuthClientID == nil || a.OAuthClientSecret == nil || a.OAuthRedirectURI == nil || a.OAuthRedirectTargetServingPath == nil {
		return false
	}
	return *a.OAuthClientID != "" && *a.OAuthClientSecret != "" && *a.OAuthRedirectURI != "" && *a.OAuthRedirectTargetServingPath != ""
}

// GetOAuthConfig returns the *oauth2.Config constructed from the given parameter.
func (a *AuthParameters) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     *a.OAuthClientID,
		ClientSecret: *a.OAuthClientSecret,
		RedirectURL:  *a.OAuthRedirectURI,
		Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform"},
		Endpoint:     google.Endpoint,
	}
}

var _ ParameterStore = (*AuthParameters)(nil)
