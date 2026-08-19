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
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/google/go-cmp/cmp"
)

func TestTimelineEvaluator(t *testing.T) {
	pool := khifilev6model.NewInternPool(&khifilev6model.IDGenerator{})
	node, err := structured.FromYAML(`kind: Pod
metadata:
  name: pod-sample
spec:
  containers:
  - name: nginx
`)
	if err != nil {
		t.Fatalf("failed to parse yaml node: %v", err)
	}
	sRef, err := khifilev6model.ToInternedStruct(node, pool)
	if err != nil {
		t.Fatalf("failed to intern struct: %v", err)
	}

	eval, err := NewTimelineEvaluator()
	if err != nil {
		t.Fatalf("failed to create TimelineEvaluator: %v", err)
	}
	eval.SetInternPool(pool)

	testTimeline := &TimelineData{
		ID:           1,
		Name:         "pod-sample",
		TimelineType: "Pod",
		Path: map[string]string{
			"namespace": "default",
			"kind":      "Pod",
			"name":      "pod-sample",
		},
		MaxSeverity: 2, // WARNING
		Revisions: []RevisionInfo{
			{
				ResourceBodyStructID: sRef.ID(),
				Severity:             2,
			},
		},
	}

	testCases := []struct {
		name       string
		expression string
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty expression matches all",
			expression: "",
			want:       true,
		},
		{
			name:       "match timeline name",
			expression: `name == "pod-sample"`,
			want:       true,
		},
		{
			name:       "match timeline name mismatch",
			expression: `name == "pod-other"`,
			want:       false,
		},
		{
			name:       "match helper with key and value",
			expression: `match("kind", "Pod")`,
			want:       true,
		},
		{
			name:       "match alias M with key and value",
			expression: `M("namespace", "default")`,
			want:       true,
		},
		{
			name:       "match helper with single wildcard argument",
			expression: `match("pod-sample")`,
			want:       true,
		},
		{
			name:       "match helper with list argument",
			expression: `match("kind", ["Deployment", "Pod"])`,
			want:       true,
		},
		{
			name:       "revision_body helper with wildcard",
			expression: `revision_body("nginx")`,
			want:       true,
		},
		{
			name:       "revision_body alias RB with path",
			expression: `RB("metadata.name", "pod-sample")`,
			want:       true,
		},
		{
			name:       "minSeverity helper",
			expression: `minSeverity(WARNING)`,
			want:       true,
		},
		{
			name:       "minSeverity helper higher than max",
			expression: `minSeverity(ERROR)`,
			want:       false,
		},
		{
			name:       "invalid expression syntax",
			expression: `name == `,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := eval.Compile(tc.expression)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Compile() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			got, err := eval.Evaluate(context.Background(), testTimeline)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogEvaluator(t *testing.T) {
	pool := khifilev6model.NewInternPool(&khifilev6model.IDGenerator{})
	logNode, err := structured.FromYAML(`verb: create
user:
  username: system:admin
`)
	if err != nil {
		t.Fatalf("failed to parse log yaml node: %v", err)
	}
	sRef, err := khifilev6model.ToInternedStruct(logNode, pool)
	if err != nil {
		t.Fatalf("failed to intern log struct: %v", err)
	}

	eval, err := NewLogEvaluator()
	if err != nil {
		t.Fatalf("failed to create LogEvaluator: %v", err)
	}
	eval.SetInternPool(pool)

	testLog := &LogData{
		ID:           10,
		LogType:      "k8s-audit",
		Severity:     3, // ERROR
		Summary:      "failed to schedule pod",
		BodyStructID: sRef.ID(),
	}

	testCases := []struct {
		name       string
		expression string
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty expression matches all",
			expression: "",
			want:       true,
		},
		{
			name:       "match severity",
			expression: `severity >= ERROR`,
			want:       true,
		},
		{
			name:       "match summary contains",
			expression: `summary.contains("schedule")`,
			want:       true,
		},
		{
			name:       "body helper with field path",
			expression: `body("user.username", "admin")`,
			want:       true,
		},
		{
			name:       "body alias B with wildcard",
			expression: `B("create")`,
			want:       true,
		},
		{
			name:       "body helper non-matching",
			expression: `body("user.username", "anonymous")`,
			want:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := eval.Compile(tc.expression)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Compile() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			got, err := eval.Evaluate(context.Background(), testLog)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Evaluate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateTimelineQuery(t *testing.T) {
	testCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "empty query is valid",
			query:   "",
			wantErr: false,
		},
		{
			name:    "valid timeline query",
			query:   `name == "pod-sample" && minSeverity(WARNING)`,
			wantErr: false,
		},
		{
			name:    "invalid syntax query",
			query:   `name == &&`,
			wantErr: true,
		},
		{
			name:    "invalid variable query",
			query:   `unknownVar == "test"`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTimelineQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateTimelineQuery() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateLogQuery(t *testing.T) {
	testCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "empty query is valid",
			query:   "",
			wantErr: false,
		},
		{
			name:    "valid log query",
			query:   `severity >= INFO && body("verb", "create")`,
			wantErr: false,
		},
		{
			name:    "invalid syntax query",
			query:   `severity >= `,
			wantErr: true,
		},
		{
			name:    "invalid function call",
			query:   `nonExistentFunction("test")`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLogQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateLogQuery() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
