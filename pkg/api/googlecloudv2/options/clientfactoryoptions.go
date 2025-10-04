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
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloudv2"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// ServiceAccountKey returns a googlecloudv2.ClientFactoryOptionsModifiers to use the given service account key for any projects.
func ServiceAccountKey(keyPath string) googlecloudv2.ClientFactoryOptionsModifiers {
	return func(opts []option.ClientOption, c googlecloudv2.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithCredentialsFile(keyPath))
		return opts, nil
	}
}

// ServiceAccountKeyForProject returns a googlecloudv2.ClientFactoryOptionsModifiers to use the given service account key for a specific project.
func ServiceAccountKeyForProject(keyPath string, projectID string) googlecloudv2.ClientFactoryOptionsModifiers {
	return func(opts []option.ClientOption, c googlecloudv2.ResourceContainer) ([]option.ClientOption, error) {
		if p, ok := c.(googlecloudv2.ProjectResourceContainer); ok && p.ProjectID() == projectID {
			opts = append(opts, option.WithCredentialsFile(keyPath))
		}
		return opts, nil
	}
}

// TokenSource returns a googlecloudv2.ClientFactoryOptionsModifiers to use the given oauth2.TokenSource for any projects.
func TokenSource(source oauth2.TokenSource) googlecloudv2.ClientFactoryOptionsModifiers {
	return func(opts []option.ClientOption, c googlecloudv2.ResourceContainer) ([]option.ClientOption, error) {
		opts = append(opts, option.WithTokenSource(source))
		return opts, nil
	}
}

// TokenSourceForProject returns a googlecloudv2.ClientFactoryOptionsModifiers to use the given oauth2.TokenSource for a specific project.
func TokenSourceForProject(projectID string, source oauth2.TokenSource) googlecloudv2.ClientFactoryOptionsModifiers {
	return func(opts []option.ClientOption, c googlecloudv2.ResourceContainer) ([]option.ClientOption, error) {
		if p, ok := c.(googlecloudv2.ProjectResourceContainer); ok && p.ProjectID() == projectID {
			opts = append(opts, option.WithTokenSource(source))
		}
		return opts, nil
	}
}
