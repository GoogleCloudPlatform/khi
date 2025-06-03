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

package k8s_container

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	gcp_log "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
)

func TestK8sContainerParserReceivingLogNotContainingMainMessage(t *testing.T) {
	parser := k8sContainerParser{}
	l, err := log.NewLogFromYAMLString("insertID: foo")
	if err != nil {
		t.Fatalf("unexpected error on constructing log instance\n%v", err)
	}
	l.SetFieldSetReader(&gcp_log.GCPMainMessageFieldSetReader{})
	cs := history.NewChangeSet(l)
	err = parser.Parse(context.Background(), l, cs, nil)
	if err != nil {
		t.Errorf("parser returned error\n%v", err)
	}
}
