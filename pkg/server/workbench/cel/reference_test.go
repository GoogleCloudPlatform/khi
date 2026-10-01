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

package cel

import (
	"strings"
	"testing"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/proto"
)

func TestGenerateTimelineReference(t *testing.T) {
	styleChunk := &khifilev6.TimelineStyleChunk{
		Severities: []*khifilev6.Severity{
			{Label: proto.String("Info"), Order: proto.Int32(10)},
			{Label: proto.String("Error"), Order: proto.Int32(30)},
		},
		TimelineTypes: []*khifilev6.TimelineType{
			{Label: proto.String("Pod"), Description: proto.String("Kubernetes Pod")},
			{Label: proto.String("Node"), Description: proto.String("Kubernetes Node")},
			{Label: proto.String("@cluster"), Description: proto.String("Synthetic Cluster")},
		},
	}

	content := GenerateTimelineReference(styleChunk)

	if !strings.HasSuffix(content, "\n") {
		t.Error("GenerateTimelineReference() should end with a newline")
	}
	if strings.HasSuffix(content, "\n\n") {
		t.Error("GenerateTimelineReference() should end with exactly a single trailing newline")
	}

	expectedSections := []string{
		"# Timeline CEL Reference",
		"## Variables",
		"## Timeline Path Keys",
		"## Severity Constants",
		"## Functions",
		"### `match` / `M`",
		"### `revision_body` / `RB`",
		"### `minSeverity`",
		"### `hasSeverity`",
		"## Common Patterns",
	}

	for _, sec := range expectedSections {
		if !strings.Contains(content, sec) {
			t.Errorf("GenerateTimelineReference() missing section: %s", sec)
		}
	}

	expectedPathKeys := []string{
		"`path[\"apiversion\"]`",
		"`path[\"kind\"]`",
		"`path[\"namespace\"]`",
		"`path[\"pod\"]`",
		"`path[\"node\"]`",
	}
	for _, pk := range expectedPathKeys {
		if !strings.Contains(content, pk) {
			t.Errorf("GenerateTimelineReference() missing path key: %s", pk)
		}
	}

	if strings.Contains(content, "@cluster") {
		t.Errorf("GenerateTimelineReference() should not include synthetic timeline types starting with '@'")
	}
}

func TestGenerateLogReference(t *testing.T) {
	styleChunk := &khifilev6.TimelineStyleChunk{
		Severities: []*khifilev6.Severity{
			{Label: proto.String("Info"), Order: proto.Int32(10)},
			{Label: proto.String("Warning"), Order: proto.Int32(20)},
		},
		LogTypes: []*khifilev6.LogType{
			{Label: proto.String("container"), Description: proto.String("Container Logs")},
			{Label: proto.String("k8s audit"), Description: proto.String("Kubernetes Audit Logs")},
		},
	}

	content := GenerateLogReference(styleChunk)

	if !strings.HasSuffix(content, "\n") {
		t.Error("GenerateLogReference() should end with a newline")
	}
	if strings.HasSuffix(content, "\n\n") {
		t.Error("GenerateLogReference() should end with exactly a single trailing newline")
	}

	expectedSections := []string{
		"# Log CEL Reference",
		"## Variables",
		"## Severity Constants",
		"## Registered Log Types",
		"## Functions",
		"### `body` / `B`",
		"## Common Patterns",
	}

	for _, sec := range expectedSections {
		if !strings.Contains(content, sec) {
			t.Errorf("GenerateLogReference() missing section: %s", sec)
		}
	}

	expectedLogTypes := []string{
		"`container`",
		"Container Logs",
		"`k8s audit`",
		"Kubernetes Audit Logs",
	}
	for _, lt := range expectedLogTypes {
		if !strings.Contains(content, lt) {
			t.Errorf("GenerateLogReference() missing log type entry: %s", lt)
		}
	}
}
