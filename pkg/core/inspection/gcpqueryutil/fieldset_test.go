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

package gcpqueryutil

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

func TestGCPCommonFieldSet(t *testing.T) {
	testCase := []struct {
		Name              string
		ExpectedTimestamp time.Time
		InputYAML         string
	}{
		{
			Name:              "from a standard GCP log",
			ExpectedTimestamp: time.Date(2025, time.January, 2, 1, 2, 3, 0, time.UTC),
			InputYAML: `insertId: foo
severity: INFO
timestamp: 2025-01-02T01:02:03.000Z`,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.InputYAML)
			if err != nil {
				t.Fatalf("failed to parse log from yaml: %v", err)
			}
			l.SetFieldSetReader(&GCPCommonFieldSetReader{})
			gcpCommonField, err := log.GetFieldSet(l, &log.CommonFieldSet{})
			if err != nil {
				t.Fatalf("failed to extract gcp common fields: %v", err)
			}
			if gcpCommonField.Timestamp != tc.ExpectedTimestamp {
				t.Errorf("expected timestamp: %v, got: %v", tc.ExpectedTimestamp, gcpCommonField.Timestamp)
			}
		})
	}
}
