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

package testlog

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

type sampleMockData struct {
	Name  string
	Count int
}

func TestNewMockLog(t *testing.T) {
	wantData := sampleMockData{Name: "test-item", Count: 10}
	l := NewMockLog(wantData)

	got, ok := structured.GetMock[sampleMockData](l.NodeReader)
	if !ok {
		t.Fatalf("structured.GetMock() returned ok=false")
	}
	if diff := cmp.Diff(wantData, got); diff != "" {
		t.Errorf("GetMock() mismatch (-want +got):\n%s", diff)
	}
}
