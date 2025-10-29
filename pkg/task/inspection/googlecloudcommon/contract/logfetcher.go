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

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"google.golang.org/api/iterator"
)

// LogFetcher is an interface for fetching logs from Cloud Logging with a given filter
// and sending them to a specified channel.
// The implementation must close the destination channel after the query is done.
type LogFetcher interface {
	FetchLogs(dest chan<- *loggingpb.LogEntry, ctx context.Context, filter string, container googlecloud.ResourceContainer, resourceContainers []string) error
}

// logFetcherImpl is the implementation of LogFetcher actually accessing to the Cloud Logging API.
type logFetcherImpl struct {
	factory            *googlecloud.ClientFactory
	callOptionInjector *googlecloud.CallOptionInjector
	pageSize           int32
	orderBy            string
}

// NewLogFetcher returns the instance of LogFetcher initialized with the given *googlecloud.ClientFactory.
func NewLogFetcher(clientFactory *googlecloud.ClientFactory, callOptionInjector *googlecloud.CallOptionInjector, pageSize int32) LogFetcher {
	return &logFetcherImpl{
		factory:            clientFactory,
		pageSize:           pageSize,
		orderBy:            "timestamp asc",
		callOptionInjector: callOptionInjector,
	}
}

// FetchLogs implements LogFetcher.
func (l *logFetcherImpl) FetchLogs(dest chan<- *loggingpb.LogEntry, ctx context.Context, filter string, container googlecloud.ResourceContainer, resourceContainers []string) error {
	defer close(dest)
	client, err := l.factory.LoggingClient(ctx, container)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx = l.callOptionInjector.InjectToCallContext(ctx, container)
	iter := client.ListLogEntries(ctx, &loggingpb.ListLogEntriesRequest{
		ResourceNames: resourceContainers,
		Filter:        filter,
		OrderBy:       l.orderBy,
		PageSize:      l.pageSize,
	}, googlecloud.DefaultRetryPolicy, googlecloud.NeverTimeout)

	for {
		entry, err := iter.Next()
		if err == iterator.Done {
			break
		}

		select {
		// Check the context cancel first
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}
		select {
		case dest <- entry:
		case <-ctx.Done():
			return ctx.Err()
		}

	}
	return nil
}
