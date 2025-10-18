// Copyright 2024 Google LLC
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

package gcp_test

import (
	"fmt"
	"testing"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"google.golang.org/api/iterator"
)

const testProjectID = "kubernetes-history-inspector"

func IsValidLogQuery(t *testing.T, query string) error {
	t.Helper()

	if *testflags.SkipCloudLogging {
		t.Skip("cloud logging tests are skipped")
	}

	factory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to initialize ClientFactory: %v", err)
	}
	lc, err := factory.LoggingClient(t.Context(), googlecloud.Project(testProjectID))
	if err != nil {
		t.Fatalf("failed to initialize LoggingClient: %v", err)
	}
	query = fmt.Sprintf(`%s
timestamp >= "2024-01-01T00:00:00Z"
timestamp <= "2024-01-01T00:00:01Z"`, query)

	iter := lc.ListLogEntries(t.Context(), &loggingpb.ListLogEntriesRequest{
		ResourceNames: []string{fmt.Sprintf("projects/%s", testProjectID)},
		Filter:        query,
		PageSize:      1,
	})

	_, err = iter.Next()
	if err != iterator.Done {
		return err
	}

	return nil
}
