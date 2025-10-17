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

	logging "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloudv2"
	"google.golang.org/api/iterator"
)

// LogFetcher is an interface for fetching logs from Cloud Logging with a given filter
// and sending them to a specified channel.
// The implementation must close the destination channel after the query is done.
type LogFetcher interface {
	FetchLogs(dest chan<- *loggingpb.LogEntry, ctx context.Context, filter string, resourceContainers []string) error
}

// logFetcherImpl is the implementation of LogFetcher actually accessing to the Cloud Logging API.
type logFetcherImpl struct {
	client   *logging.Client
	pageSize int32
	orderBy  string
}

// NewLogFetcher returns the instance of LogFetcher initialized with the given logging client.
func NewLogFetcher(client *logging.Client, pageSize int32) LogFetcher {
	return &logFetcherImpl{
		client:   client,
		pageSize: pageSize,
		orderBy:  "timestamp asc",
	}
}

// FetchLogs implements LogFetcher.
func (l *logFetcherImpl) FetchLogs(dest chan<- *loggingpb.LogEntry, ctx context.Context, filter string, resourceContainers []string) error {
	defer close(dest)

	iter := l.client.ListLogEntries(ctx, &loggingpb.ListLogEntriesRequest{
		ResourceNames: resourceContainers,
		Filter:        filter,
		OrderBy:       l.orderBy,
		PageSize:      l.pageSize,
	}, googlecloudv2.DefaultRetryPolicy, googlecloudv2.NeverTimeout)

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
