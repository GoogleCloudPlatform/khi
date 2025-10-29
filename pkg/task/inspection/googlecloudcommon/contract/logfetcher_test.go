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

package googlecloudcommon_contract

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
)

func TestLogFetcherImpl_FetchLogs(t *testing.T) {
	if *testflags.SkipCloudLogging {
		t.Skip()
		return
	}

	cf, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to instanciate client factory: %v", err)
	}
	fetcher := logFetcherImpl{
		factory:            cf,
		pageSize:           1,
		orderBy:            "timestamp desc", // just need the latest log. getting oldest log takes longer time.
		callOptionInjector: googlecloud.NewCallOptionInjector(),
	}

	ctx, cancel := context.WithCancel(t.Context())
	destChan := make(chan *loggingpb.LogEntry)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		select {
		// Test time out is 30 sec by default and getting a single log for 20 sec timeout must be fine.
		case <-time.After(20 * time.Second):
			t.Errorf("no logs returned for the first 20 sec")
		case _, ok := <-destChan:
			if !ok {
				t.Errorf("channel closed before receiving any response")
			}
			cancel() // this test just receive a log. Cancel context after receiving one.
		}
	}()

	err = fetcher.FetchLogs(destChan, ctx, "", googlecloud.Project("kubernetes-history-inspector"), []string{"projects/kubernetes-history-inspector"})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("failed to fetch logs:%v", err)
	}
	wg.Wait()
}

func TestLogFetcherImpl_FetchLogsIsCancellable(t *testing.T) {
	if *testflags.SkipCloudLogging {
		t.Skip()
		return
	}

	cf, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to instanciate client factory: %v", err)
	}
	fetcher := logFetcherImpl{
		factory:            cf,
		pageSize:           1000,
		callOptionInjector: googlecloud.NewCallOptionInjector(),
	}

	fetchLogFinished := make(chan struct{})
	destChan := make(chan *loggingpb.LogEntry)
	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		<-time.After(100 * time.Millisecond)
		cancel()
	}()

	go func() {
		select {
		case <-time.After(500 * time.Millisecond):
			t.Errorf("the request wasn't cancelled after 500ms")
		case <-fetchLogFinished:
			return
		}
	}()

	err = fetcher.FetchLogs(destChan, ctx, "", googlecloud.Project("kubernetes-history-inspector"), []string{"projects/kubernetes-history-inspector"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the request wasn't ended with canceled but got %v", err)
	}
	close(fetchLogFinished)
}
