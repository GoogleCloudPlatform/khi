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
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
)

func IsValidLogQuery(t *testing.T, query string) error {
	t.Helper()

	if *testflags.SkipCloudLogging {
		t.Skip("cloud logging tests are skipped")
	}
	accessToken, found := os.LookupEnv("GCP_ACCESS_TOKEN")
	if found {
		parameters.Auth.AccessToken = &accessToken
	}
	gcpApi, err := api.DefaultGCPClientFactory.NewClient()
	if err != nil {
		return err
	}
	query = fmt.Sprintf(`%s
timestamp >= "2024-01-01T00:00:00Z"
timestamp <= "2024-01-01T00:00:01Z"`, query)

	return gcpApi.ListLogEntries(context.Background(), []string{"projects/kubernetes-history-inspector"}, query, make(chan any))
}
