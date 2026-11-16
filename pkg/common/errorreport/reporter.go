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

package errorreport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"

	"cloud.google.com/go/errorreporting"
	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/khi/pkg/common/constants"
)

// DefaultErrorReporter is the default reporter to be used for reporting an error on panic.
var DefaultErrorReporter *Reporter = &Reporter{
	metadata: map[string]string{},
	writer:   &ConsoleReportWriter{},
}

var (
	_ ReportWriter = &ConsoleReportWriter{}
	_ ReportWriter = &CloudErrorReportWriter{}
)

// Reporter records errors with attached metadata using a ReportWriter.
type Reporter struct {
	metadata     map[string]string
	metadataLock sync.RWMutex
	writer       ReportWriter
}

// ReportWriter is an interface to send an error.
// The destination can be only stdout or a backend for collecting errors.
type ReportWriter interface {
	// WriteReportSync reports the given error.
	WriteReportSync(ctx context.Context, err error)
}

// ConsoleReportWriter reports error through stderr.
type ConsoleReportWriter struct {
}

// CloudErrorReportWriter sends error over Cloud Error Reporting on GCP.
type CloudErrorReportWriter struct {
	client *errorreporting.Client
}

// NewReporter returns a new Reporter with the given ReportWriter.
func NewReporter(reportWriter ReportWriter) *Reporter {
	return &Reporter{
		metadata: map[string]string{},
		writer:   reportWriter,
	}
}

// GetMetadata returns a copy of the current metadata map.
func (r *Reporter) GetMetadata() map[string]string {
	defer r.metadataLock.RUnlock()
	r.metadataLock.RLock()
	return maps.Clone(r.metadata)
}

// SetMetadataEntry sets a metadata key-value pair included in subsequent error reports.
func (r *Reporter) SetMetadataEntry(key string, value string) {
	r.metadataLock.Lock()
	defer r.metadataLock.Unlock()
	r.metadata[key] = value
}

// ReportSync reports the error using the report writer.
func (r *Reporter) ReportSync(ctx context.Context, err error) {
	r.writer.WriteReportSync(ctx, r.getErrorMessageWithMetadata(err))
}

func (r *Reporter) getErrorMessageWithMetadata(err error) error {
	metadata := r.GetMetadata()
	message := err.Error()
	if len(metadata) > 0 {
		// convert metadata map to bullet point list string
		metadataList := ""
		keys := maps.Keys(metadata)
		sortedKeys := slices.Sorted(keys)
		for _, key := range sortedKeys {
			metadataList += fmt.Sprintf("    * %s: %s\n", key, metadata[key])
		}
		message = fmt.Sprintf("%s\n  Metadata:\n%v", message, metadataList)
	}
	return errors.New(message)
}

// WriteReportSync implements ReportWriter.
func (r *ConsoleReportWriter) WriteReportSync(ctx context.Context, err error) {
	slog.ErrorContext(ctx, err.Error())
}

// NewCloudErrorReportWriter returns a new instance of CloudErrorReporter.
func NewCloudErrorReportWriter(projectId string, apiKey string) (*CloudErrorReportWriter, error) {
	client, err := errorreporting.NewClient(context.Background(), projectId, errorreporting.Config{
		ServiceName:    "kubernetes-history-inspector",
		ServiceVersion: constants.VERSION,
	}, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &CloudErrorReportWriter{client: client}, nil
}

// WriteReportSync implements Reporter.
func (c *CloudErrorReportWriter) WriteReportSync(ctx context.Context, err error) {
	writeError := c.client.ReportSync(ctx, errorreporting.Entry{
		Error: err,
	})
	if writeError != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Failed to write the error to Cloud Error Reporting due to an error:%v", writeError))
	}
}
