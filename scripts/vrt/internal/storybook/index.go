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

package storybook

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	indexFileName                 = "index.json"
	staticServerReadHeaderTimeout = 10 * time.Second
	staticServerShutdownTimeout   = 5 * time.Second
)

// StoryEntry describes a single renderable Storybook story parsed from index.json.
type StoryEntry struct {
	// ID is the unique Storybook story identifier used in iframe.html?id=...
	ID string `json:"id"`
	// Title is the Storybook hierarchy title (e.g., "Timeline/Main/TimelineChart").
	Title string `json:"title"`
	// Name is the human-readable story variant name (e.g., "Default").
	Name string `json:"name"`
	// ImportPath is the relative source path of the story file.
	ImportPath string `json:"importPath"`
	// ComponentPath is the relative source path of the component under test.
	ComponentPath string `json:"componentPath,omitempty"`
	// Tags contains Storybook tags declared on the story.
	Tags []string `json:"tags,omitempty"`
}

// StaticServer serves a built storybook-static directory on a loopback HTTP port.
type StaticServer struct {
	server   *http.Server
	listener net.Listener
	baseURL  string
}

type indexDocument struct {
	Version int                       `json:"v"`
	Entries map[string]rawIndexRecord `json:"entries"`
}

type rawIndexRecord struct {
	Type          string   `json:"type"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Name          string   `json:"name"`
	ImportPath    string   `json:"importPath"`
	ComponentPath string   `json:"componentPath"`
	Tags          []string `json:"tags"`
}

// LoadStories reads index.json inside storybookDir and returns deterministically sorted stories matching filterPattern.
func LoadStories(storybookDir string, filterPattern string) ([]StoryEntry, error) {
	indexPath := filepath.Join(storybookDir, indexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storybook index %s: %w", indexPath, err)
	}

	var doc indexDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse storybook index %s: %w", indexPath, err)
	}

	matcher, err := compileFilter(filterPattern)
	if err != nil {
		return nil, err
	}

	stories := make([]StoryEntry, 0, len(doc.Entries))
	for key, rec := range doc.Entries {
		if rec.Type != "story" {
			continue
		}
		id := rec.ID
		if id == "" {
			id = key
		}
		entry := StoryEntry{
			ID:            id,
			Title:         rec.Title,
			Name:          rec.Name,
			ImportPath:    rec.ImportPath,
			ComponentPath: rec.ComponentPath,
			Tags:          rec.Tags,
		}
		if matchesFilter(matcher, entry) {
			stories = append(stories, entry)
		}
	}

	slices.SortFunc(stories, func(a, b StoryEntry) int {
		return strings.Compare(a.ID, b.ID)
	})
	return stories, nil
}

// StartStaticServer binds a local HTTP file server serving storybookDir on an ephemeral loopback port.
func StartStaticServer(storybookDir string) (*StaticServer, error) {
	info, err := os.Stat(storybookDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access storybook directory %s: %w", storybookDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("storybook path %s is not a directory", storybookDir)
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind loopback HTTP server: %w", err)
	}

	fileServer := http.FileServer(http.Dir(storybookDir))
	srv := &http.Server{
		Handler:           fileServer,
		ReadHeaderTimeout: staticServerReadHeaderTimeout,
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	return &StaticServer{
		server:   srv,
		listener: listener,
		baseURL:  "http://" + listener.Addr().String(),
	}, nil
}

// BaseURL returns the base loopback URL (without trailing slash) of the running static server.
func (s *StaticServer) BaseURL() string {
	return s.baseURL
}

// Close shuts down the static HTTP server gracefully.
func (s *StaticServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), staticServerShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func compileFilter(pattern string) (*regexp.Regexp, error) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return nil, nil
	}
	re, err := regexp.Compile("(?i)" + trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid story filter regex %q: %w", pattern, err)
	}
	return re, nil
}

func matchesFilter(re *regexp.Regexp, entry StoryEntry) bool {
	if re == nil {
		return true
	}
	return re.MatchString(entry.ID) ||
		re.MatchString(entry.Title) ||
		re.MatchString(entry.Name) ||
		re.MatchString(entry.ImportPath)
}
