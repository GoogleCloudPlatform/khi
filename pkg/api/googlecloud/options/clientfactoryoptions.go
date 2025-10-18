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

package options

import (
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/oauth"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

func fromClientFactoryOptionsModifier(modifier googlecloud.ClientFactoryOptionsModifiers) googlecloud.ClientFactoryOption {
	return func(s *googlecloud.ClientFactory) error {
		s.ClientOptions = append(s.ClientOptions, modifier)
		return nil
	}
}

// ServiceAccountKey returns a googlecloud.ClientFactoryOption to use the given service account key for any projects.
func ServiceAccountKey(keyPath string) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithCredentialsFile(keyPath))
		return opts, nil
	})
}

// ServiceAccountKeyForProject returns a googlecloud.ClientFactoryOption to use the given service account key for a specific project.
func ServiceAccountKeyForProject(keyPath string, projectID string) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		if p, ok := c.(googlecloud.ProjectResourceContainer); ok && p.ProjectID() == projectID {
			opts = append(opts, option.WithCredentialsFile(keyPath))
		}
		return opts, nil
	})
}

// TokenSource returns a googlecloud.ClientFactoryOption to use the given oauth2.TokenSource for any projects.
func TokenSource(source oauth2.TokenSource) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithTokenSource(source))
		return opts, nil
	})
}

// TokenSourceForProject returns a googlecloud.ClientFactoryOption to use the given oauth2.TokenSource for a specific project.
func TokenSourceForProject(projectID string, source oauth2.TokenSource) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		if p, ok := c.(googlecloud.ProjectResourceContainer); ok && p.ProjectID() == projectID {
			opts = append(opts, option.WithTokenSource(source))
		}
		return opts, nil
	})
}

// OAuth returns a googlecloud.ClientFactoryOption that configures the client to use
// the oauth2.TokenSource provided by the given oauth.OAuthServer.
// This allows the Google Cloud client to obtain access tokens via an OAuth 2.0 flow managed by the OAuthServer.
func OAuth(server *oauth.OAuthServer) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithTokenSource(server.TokenSource()))
		return opts, nil
	})
}

// QuotaProject returns a googlecloud.ClientFactoryOption that configures the client to use the specified quota project.
func QuotaProject(projectID string) googlecloud.ClientFactoryOption {
	return fromClientFactoryOptionsModifier(func(opts []option.ClientOption, c googlecloud.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithQuotaProject(projectID))
		return opts, nil
	})
}
