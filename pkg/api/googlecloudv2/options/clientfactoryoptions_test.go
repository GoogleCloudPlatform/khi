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

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloudv2"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloudv2/oauth"
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

// GetType implements googlecloudv2.ResourceContainer.
func (c *nonProjectContainer) GetType() googlecloudv2.ResourceContainerType {
	return googlecloudv2.ResourceContainerInvalid
}

// Ensure nonProjectContainer implements the interface.
var _ googlecloudv2.ResourceContainer = (*nonProjectContainer)(nil)

func TestServiceAccountKey(t *testing.T) {
	modifier := ServiceAccountKey("test-key-path")
	container := googlecloudv2.Project("any-project")

	opts, err := modifier([]option.ClientOption{}, container)
	if err != nil {
		t.Fatalf("modifier returned an unexpected error: %v", err)
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
		container     googlecloudv2.ResourceContainer
		expectsOption bool
	}{
		{
			name:          "Matching project",
			container:     googlecloudv2.Project(projectID),
			expectsOption: true,
		},
		{
			name:          "Non-matching project",
			container:     googlecloudv2.Project("other-project"),
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
			modifier := ServiceAccountKeyForProject(keyPath, projectID)
			opts, err := modifier([]option.ClientOption{}, tc.container)

			if err != nil {
				t.Fatalf("modifier returned an unexpected error: %v", err)
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
	modifier := TokenSource(source)
	container := googlecloudv2.Project("any-project")

	opts, err := modifier([]option.ClientOption{}, container)
	if err != nil {
		t.Fatalf("modifier returned an unexpected error: %v", err)
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
		container     googlecloudv2.ResourceContainer
		expectsOption bool
	}{
		{
			name:          "Matching project",
			container:     googlecloudv2.Project(projectID),
			expectsOption: true,
		},
		{
			name:          "Non-matching project",
			container:     googlecloudv2.Project("other-project"),
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
			modifier := TokenSourceForProject(projectID, source)
			opts, err := modifier([]option.ClientOption{}, tc.container)

			if err != nil {
				t.Fatalf("modifier returned an unexpected error: %v", err)
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
	modifier := OAuth(server)
	container := googlecloudv2.Project("any-project")
	opts, err := modifier([]option.ClientOption{}, container)
	if err != nil {
		t.Fatalf("modifier returned an unexpected error: %v", err)
	}
	if len(opts) != 1 {
		t.Errorf("Expected 1 option to be added, but got %d", len(opts))
	}
}
