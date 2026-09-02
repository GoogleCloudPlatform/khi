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

package log

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
)

func TestNewLog(t *testing.T) {
	gen := id.NewGenerator()
	node, err := structured.FromYAML("foo: bar")
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	reader := structured.NewNodeReader(node)
	l := NewLog(gen, reader)

	if l.NodeReader != reader {
		t.Errorf("NewLog() NodeReader mismatch (-want +got):\n%s", cmp.Diff(reader, l.NodeReader))
	}
	if l.ID == 0 {
		t.Errorf("NewLog() expected non-zero ID, got 0")
	}
}

func TestNewLogWithTimestamp(t *testing.T) {
	gen := id.NewGenerator()
	node, err := structured.FromYAML("foo: bar")
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	reader := structured.NewNodeReader(node)
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLogWithTimestamp(gen, reader, testTime)

	if !l.Timestamp.Equal(testTime) {
		t.Errorf("NewLogWithTimestamp() Timestamp mismatch (-want +got):\n%s", cmp.Diff(testTime, l.Timestamp))
	}
	if l.ID == 0 {
		t.Errorf("NewLogWithTimestamp() expected non-zero ID, got 0")
	}
}

func TestNewLogFromYAMLString(t *testing.T) {
	gen := id.NewGenerator()
	testCases := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name:    "valid yaml",
			yaml:    "foo: bar",
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			yaml:    ":\ninvalid",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewLogFromYAMLString(gen, tc.yaml)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewLogFromYAMLString() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got == nil {
				t.Errorf("NewLogFromYAMLString() returned nil log on valid yaml")
			}
		})
	}
}
