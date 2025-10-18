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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/oauth"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// mockTokenSource is a simple implementation of oauth2.TokenSource for testing.
type mockTokenSource struct{}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{},
		nil
}

// nonProjectContainer is a dummy implementation of ResourceContainer for testing.
type nonProjectContainer struct{}

// GetType implements googlecloud.ResourceContainer.
func (c *nonProjectContainer) GetType() googlecloud.ResourceContainerType {
	return googlecloud.ResourceContainerInvalid
}

// Ensure nonProjectContainer implements the interface.
var _ googlecloud.ResourceContainer = (*nonProjectContainer)(nil)

func TestServiceAccountKey(t *testing.T) {
	optionFunc := ServiceAccountKey("test-key-path")
	container := googlecloud.Project("any-project")
	clientFactory := googlecloud.ClientFactory{}
	err := optionFunc(&clientFactory)
	if err != nil {
		t.Errorf("optionFunc returned an unexpected error: %v", err)
	}
	clientOpts := clientFactory.ClientOptions
	if len(clientOpts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(clientOpts))
	}

	opts, err := clientOpts[0]([]option.ClientOption{}, container)
	if err != nil {
		t.Errorf("client option returned an unexpected error: %v", err)
	}

	if len(opts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(opts))
	}
}

func TestServiceAccountKeyForProject(t *testing.T) {
	const keyPath = "test-key-path"
	const projectID = "target-project"

	testCases := []struct {
		name          string
		container     googlecloud.ResourceContainer
		expectsOption bool
	}{
		{
			name:          "Matching project",
			container:     googlecloud.Project(projectID),
			expectsOption: true,
		},
		{
			name:          "Non-matching project",
			container:     googlecloud.Project("other-project"),
			expectsOption: false,
		},
		{
			name:          "Non-project container",
			container:     &nonProjectContainer{},
			expectsOption: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			optionFunc := ServiceAccountKeyForProject(keyPath, projectID)

			clientFactory := googlecloud.ClientFactory{}
			err := optionFunc(&clientFactory)
			if err != nil {
				t.Errorf("optionFunc returned an unexpected error: %v", err)
			}
			clientOpts := clientFactory.ClientOptions
			if len(clientOpts) != 1 {
				t.Errorf("Expected 1 option to be added, but got %d", len(clientOpts))
			}

			opts, err := clientOpts[0]([]option.ClientOption{}, tc.container)
			if err != nil {
				t.Errorf("client option returned an unexpected error: %v", err)
			}

			expectedLen := 0
			if tc.expectsOption {
				expectedLen = 1
			}

			if len(opts) != expectedLen {
				t.Errorf("Expected %d options, but got %d", expectedLen, len(opts))
			}
		})
	}
}

func TestTokenSource(t *testing.T) {
	source := &mockTokenSource{}
	optionFunc := TokenSource(source)
	container := googlecloud.Project("any-project")

	clientFactory := googlecloud.ClientFactory{}
	err := optionFunc(&clientFactory)
	if err != nil {
		t.Errorf("optionFunc returned an unexpected error: %v", err)
	}
	clientOpts := clientFactory.ClientOptions
	if len(clientOpts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(clientOpts))
	}

	opts, err := clientOpts[0]([]option.ClientOption{}, container)
	if err != nil {
		t.Errorf("client option returned an unexpected error: %v", err)
	}

	if len(opts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(opts))
	}
}

func TestTokenSourceForProject(t *testing.T) {
	const projectID = "target-project"
	source := &mockTokenSource{}

	testCases := []struct {
		name          string
		container     googlecloud.ResourceContainer
		expectsOption bool
	}{
		{
			name:          "Matching project",
			container:     googlecloud.Project(projectID),
			expectsOption: true,
		},
		{
			name:          "Non-matching project",
			container:     googlecloud.Project("other-project"),
			expectsOption: false,
		},
		{
			name:          "Non-project container",
			container:     &nonProjectContainer{},
			expectsOption: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			optionFunc := TokenSourceForProject(projectID, source)
			clientFactory := googlecloud.ClientFactory{}
			err := optionFunc(&clientFactory)
			if err != nil {
				t.Errorf("optionFunc returned an unexpected error: %v", err)
			}
			clientOpts := clientFactory.ClientOptions
			if len(clientOpts) != 1 {
				t.Errorf("Expected 1 option to be added, but got %d", len(clientOpts))
			}

			opts, err := clientOpts[0]([]option.ClientOption{}, tc.container)
			if err != nil {
				t.Errorf("client option returned an unexpected error: %v", err)
			}

			expectedLen := 0
			if tc.expectsOption {
				expectedLen = 1
			}

			if len(opts) != expectedLen {
				t.Errorf("Expected %d options, but got %d", expectedLen, len(opts))
			}
		})
	}
}

func TestOAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	conf := &oauth2.Config{}
	server := oauth.NewOAuthServer(engine, conf, "/callback", "-suffix")
	optionFunc := OAuth(server)
	container := googlecloud.Project("any-project")
	clientFactory := googlecloud.ClientFactory{}
	err := optionFunc(&clientFactory)
	if err != nil {
		t.Errorf("optionFunc returned an unexpected error: %v", err)
	}
	clientOpts := clientFactory.ClientOptions
	if len(clientOpts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(clientOpts))
	}

	opts, err := clientOpts[0]([]option.ClientOption{}, container)
	if err != nil {
		t.Errorf("client option returned an unexpected error: %v", err)
	}
	if len(opts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(opts))
	}
}
