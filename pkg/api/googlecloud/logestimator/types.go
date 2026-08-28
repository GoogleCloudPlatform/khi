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

package logestimator

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/logging/apiv2/loggingpb"
)

// EstimatedCountPreset represents a predefined rough estimation tier.
type EstimatedCountPreset int

const (
	// EstimatedCountPresetNone indicates that standard numeric estimation should be performed.
	EstimatedCountPresetNone EstimatedCountPreset = iota

	// EstimatedCountPresetFew indicates that the query matches only a small number of logs.
	EstimatedCountPresetFew
)

// String returns the string representation of the preset.
func (p EstimatedCountPreset) String() string {
	switch p {
	case EstimatedCountPresetFew:
		return "few"
	default:
		return ""
	}
}

// StructuredLogQuery represents a decomposed Cloud Logging and Cloud Monitoring query.
type StructuredLogQuery struct {
	// ResourceTypes specifies the resource.type strings (e.g. ["k8s_container"] or ["gke_cluster", "gke_nodepool"]).
	ResourceTypes []string

	// IgnoreMetricsResourceType specifies the resource.type strings that should NOT be queried via Cloud Monitoring metrics
	// (e.g. resource types like "gke_nodepool" that are valid in Cloud Logging but do not exist as MonitoredResourceDescriptors in Cloud Monitoring).
	IgnoreMetricsResourceType []string

	// Filters are the individual filter matchers for this query.
	Filters []LoggingMonitoringMatcher

	// Incomplete indicates that the query is missing required parameters (e.g. ProjectID or ClusterName)
	// and should not trigger log volume estimation.
	Incomplete bool

	// Preset specifies a predefined rough estimation tier (e.g. EstimatedCountPresetFew)
	// that bypasses Cloud Monitoring and probe queries.
	Preset EstimatedCountPreset
}

// GenerateCloudLoggingQuery generates the Cloud Logging filter string.
func (q *StructuredLogQuery) GenerateCloudLoggingQuery() string {
	var parts []string

	// Resource type
	if len(q.ResourceTypes) == 1 {
		parts = append(parts, fmt.Sprintf(`resource.type="%s"`, q.ResourceTypes[0]))
	} else if len(q.ResourceTypes) > 1 {
		quoted := quoteStrings(q.ResourceTypes)
		parts = append(parts, fmt.Sprintf(`resource.type=(%s)`, strings.Join(quoted, " OR ")))
	}

	for _, matcher := range q.Filters {
		if matcher != nil {
			snippet := matcher.ToLoggingQuery()
			if snippet != "" {
				parts = append(parts, snippet)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// AllFiltersSupportMetrics returns true if all matchers in Filters can be natively evaluated in Cloud Monitoring
// and no resource types are excluded from metrics querying.
func (q *StructuredLogQuery) AllFiltersSupportMetrics() bool {
	if len(q.IgnoreMetricsResourceType) > 0 {
		return false
	}
	for _, matcher := range q.Filters {
		if matcher != nil && !matcher.SupportMetrics() {
			return false
		}
	}
	return true
}

// GenerateBaseLoggingQuery generates the Cloud Logging filter string containing only matchers supported by Cloud Monitoring.
func (q *StructuredLogQuery) GenerateBaseLoggingQuery() string {
	var parts []string

	ignoreSet := make(map[string]struct{}, len(q.IgnoreMetricsResourceType))
	for _, ignore := range q.IgnoreMetricsResourceType {
		ignoreSet[ignore] = struct{}{}
	}
	var metricTypes []string
	for _, resType := range q.ResourceTypes {
		if _, ignored := ignoreSet[resType]; !ignored {
			metricTypes = append(metricTypes, resType)
		}
	}

	// Resource type
	if len(metricTypes) == 1 {
		parts = append(parts, fmt.Sprintf(`resource.type="%s"`, metricTypes[0]))
	} else if len(metricTypes) > 1 {
		quoted := quoteStrings(metricTypes)
		parts = append(parts, fmt.Sprintf(`resource.type=(%s)`, strings.Join(quoted, " OR ")))
	}

	for _, matcher := range q.Filters {
		if matcher != nil && matcher.SupportMetrics() {
			snippet := matcher.ToLoggingQuery()
			if snippet != "" {
				parts = append(parts, snippet)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// GenerateMonitoringMetricFilters generates the filter strings for logging.googleapis.com/log_entry_count in Cloud Monitoring (one per resource type).
func (q *StructuredLogQuery) GenerateMonitoringMetricFilters() []string {
	ignoreSet := make(map[string]struct{}, len(q.IgnoreMetricsResourceType))
	for _, ignore := range q.IgnoreMetricsResourceType {
		ignoreSet[ignore] = struct{}{}
	}

	var types []string
	for _, resType := range q.ResourceTypes {
		if _, ignored := ignoreSet[resType]; !ignored {
			types = append(types, resType)
		}
	}
	if len(types) == 0 {
		if len(q.ResourceTypes) == 0 {
			types = []string{""}
		} else {
			return nil
		}
	}

	var metricFilterParts []string
	for _, matcher := range q.Filters {
		if matcher != nil && matcher.SupportMetrics() {
			snippet := matcher.ToMonitoringFilter()
			if snippet != "" {
				metricFilterParts = append(metricFilterParts, snippet)
			}
		}
	}

	var filters []string
	for _, resType := range types {
		parts := []string{`metric.type = "logging.googleapis.com/log_entry_count"`}
		if resType != "" {
			parts = append(parts, fmt.Sprintf(`resource.type = "%s"`, resType))
		}
		parts = append(parts, metricFilterParts...)
		filters = append(filters, strings.Join(parts, " AND "))
	}

	return filters
}

// LogProbeClient is an interface for probing Cloud Logging.
type LogProbeClient interface {
	FetchProbe(ctx context.Context, filter string, pageSize int32) ([]*loggingpb.LogEntry, error)
}

// EstimateResult represents the volume estimate.
type EstimateResult struct {
	// MetricCount is the base count obtained from Cloud Monitoring metric.
	MetricCount int64 `json:"metricCount"`

	// EstimatedCount is the estimated total count after applying custom filters.
	EstimatedCount int64 `json:"estimatedCount"`

	// CustomFilterRatio is the ratio r (0.0 to 1.0) applied to MetricCount.
	CustomFilterRatio float64 `json:"customFilterRatio"`

	// IsExact is true if no custom filter ratio estimation was required.
	IsExact bool `json:"isExact"`

	// Preset indicates a rough estimation tier if defined on the query.
	Preset EstimatedCountPreset `json:"preset,omitempty"`
}
