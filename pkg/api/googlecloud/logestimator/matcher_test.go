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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/google/go-cmp/cmp"
)

func TestResourceLabelMatchers(t *testing.T) {
	tests := []struct {
		name               string
		matcher            LoggingMonitoringMatcher
		wantLoggingQuery   string
		wantMonitoring     string
		wantSupportMetrics bool
	}{
		{
			name:               "ResourceLabel with Exact",
			matcher:            ResourceLabel("cluster_name", Exact("my-cluster")),
			wantLoggingQuery:   `resource.labels.cluster_name="my-cluster"`,
			wantMonitoring:     `resource.labels.cluster_name = "my-cluster"`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with OneOf multiple",
			matcher:            ResourceLabel("namespace_name", OneOf("default", "kube-system")),
			wantLoggingQuery:   `resource.labels.namespace_name=("default" OR "kube-system")`,
			wantMonitoring:     `resource.labels.namespace_name = one_of("default", "kube-system")`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with OneOf single",
			matcher:            ResourceLabel("namespace_name", OneOf("default")),
			wantLoggingQuery:   `resource.labels.namespace_name="default"`,
			wantMonitoring:     `resource.labels.namespace_name = "default"`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with NoneOf multiple",
			matcher:            ResourceLabel("namespace_name", NoneOf("kube-system", "istio-system")),
			wantLoggingQuery:   `-resource.labels.namespace_name=("kube-system" OR "istio-system")`,
			wantMonitoring:     `resource.labels.namespace_name != "kube-system" AND resource.labels.namespace_name != "istio-system"`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with NoneOf single",
			matcher:            ResourceLabel("namespace_name", NoneOf("kube-system")),
			wantLoggingQuery:   `-resource.labels.namespace_name="kube-system"`,
			wantMonitoring:     `resource.labels.namespace_name != "kube-system"`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with ContainsAny multiple",
			matcher:            ResourceLabel("pod_name", ContainsAny("nginx", "redis")),
			wantLoggingQuery:   `resource.labels.pod_name:("nginx" OR "redis")`,
			wantMonitoring:     `(resource.labels.pod_name = has_substring("nginx") OR resource.labels.pod_name = has_substring("redis"))`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with ContainsAny single",
			matcher:            ResourceLabel("pod_name", ContainsAny("nginx")),
			wantLoggingQuery:   `resource.labels.pod_name:"nginx"`,
			wantMonitoring:     `resource.labels.pod_name = has_substring("nginx")`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with NotContainsAny multiple",
			matcher:            ResourceLabel("pod_name", NotContainsAny("nginx", "redis")),
			wantLoggingQuery:   `-resource.labels.pod_name:("nginx" OR "redis")`,
			wantMonitoring:     `(resource.labels.pod_name != has_substring("nginx") AND resource.labels.pod_name != has_substring("redis"))`,
			wantSupportMetrics: true,
		},
		{
			name:               "ResourceLabel with NotContainsAny single",
			matcher:            ResourceLabel("pod_name", NotContainsAny("nginx")),
			wantLoggingQuery:   `-resource.labels.pod_name:"nginx"`,
			wantMonitoring:     `resource.labels.pod_name != has_substring("nginx")`,
			wantSupportMetrics: true,
		},
		{
			name: "ResourceLabel with FromSetFilter additive exact",
			matcher: ResourceLabel("namespace_name", FromSetFilter(&gcpqueryutil.SetFilterParseResult{
				Additives: []string{"prod", "staging"},
			}, false)),
			wantLoggingQuery:   `resource.labels.namespace_name=("prod" OR "staging")`,
			wantMonitoring:     `resource.labels.namespace_name = one_of("prod", "staging")`,
			wantSupportMetrics: true,
		},
		{
			name: "ResourceLabel with FromSetFilter subtractive substring",
			matcher: ResourceLabel("pod_name", FromSetFilter(&gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"canary", "test"},
			}, true)),
			wantLoggingQuery:   `-resource.labels.pod_name:("canary" OR "test")`,
			wantMonitoring:     `(resource.labels.pod_name != has_substring("canary") AND resource.labels.pod_name != has_substring("test"))`,
			wantSupportMetrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.SupportMetrics(); got != tt.wantSupportMetrics {
				t.Errorf("SupportMetrics() = %v, want %v", got, tt.wantSupportMetrics)
			}

			gotLogging := tt.matcher.ToLoggingQuery()
			if diff := cmp.Diff(tt.wantLoggingQuery, gotLogging); diff != "" {
				t.Errorf("ToLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			gotMonitoring := tt.matcher.ToMonitoringFilter()
			if diff := cmp.Diff(tt.wantMonitoring, gotMonitoring); diff != "" {
				t.Errorf("ToMonitoringFilter() filter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogIDMatchers(t *testing.T) {
	tests := []struct {
		name               string
		matcher            LoggingMonitoringMatcher
		wantLoggingQuery   string
		wantMonitoring     string
		wantSupportMetrics bool
	}{
		{
			name:               "LogID with Exact",
			matcher:            LogID(Exact("events")),
			wantLoggingQuery:   `LOG_ID("events")`,
			wantMonitoring:     `metric.labels.log = "events"`,
			wantSupportMetrics: true,
		},
		{
			name:               "LogID with OneOf multiple",
			matcher:            LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
			wantLoggingQuery:   `(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))`,
			wantMonitoring:     `metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			wantSupportMetrics: true,
		},
		{
			name:               "LogID with OneOf single",
			matcher:            LogID(OneOf("events")),
			wantLoggingQuery:   `LOG_ID("events")`,
			wantMonitoring:     `metric.labels.log = "events"`,
			wantSupportMetrics: true,
		},
		{
			name:               "LogID with NoneOf multiple",
			matcher:            LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
			wantLoggingQuery:   `-LOG_ID("server-accesslog-stackdriver")` + "\n" + `-LOG_ID("client-accesslog-stackdriver")`,
			wantMonitoring:     `metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver"`,
			wantSupportMetrics: true,
		},
		{
			name:               "LogID with NoneOf single",
			matcher:            LogID(NoneOf("events")),
			wantLoggingQuery:   `-LOG_ID("events")`,
			wantMonitoring:     `metric.labels.log != "events"`,
			wantSupportMetrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.SupportMetrics(); got != tt.wantSupportMetrics {
				t.Errorf("SupportMetrics() = %v, want %v", got, tt.wantSupportMetrics)
			}

			gotLogging := tt.matcher.ToLoggingQuery()
			if diff := cmp.Diff(tt.wantLoggingQuery, gotLogging); diff != "" {
				t.Errorf("ToLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			gotMonitoring := tt.matcher.ToMonitoringFilter()
			if diff := cmp.Diff(tt.wantMonitoring, gotMonitoring); diff != "" {
				t.Errorf("ToMonitoringFilter() filter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCustomFilterMatcher(t *testing.T) {
	customQuery := `protoPayload.methodName: ("create" OR "update")`
	m := CustomFilter(customQuery)

	if m.SupportMetrics() {
		t.Errorf("SupportMetrics() = true, want false")
	}

	if diff := cmp.Diff(customQuery, m.ToLoggingQuery()); diff != "" {
		t.Errorf("ToLoggingQuery() mismatch (-want +got):\n%s", diff)
	}

	filter := m.ToMonitoringFilter()
	if filter != "" {
		t.Errorf("ToMonitoringFilter() filter = %q, want empty", filter)
	}

	// Empty custom filter returns nil
	if nilMatcher := CustomFilter("   "); nilMatcher != nil {
		t.Errorf("CustomFilter(\"   \") = %v, want nil", nilMatcher)
	}
}

func TestCommentMatcher(t *testing.T) {
	testCases := []struct {
		name               string
		comment            string
		wantNil            bool
		wantLoggingQuery   string
		wantMonitoring     string
		wantSupportMetrics bool
	}{
		{
			name:               "standard comment",
			comment:            "this is a comment",
			wantNil:            false,
			wantLoggingQuery:   "-- this is a comment",
			wantMonitoring:     "",
			wantSupportMetrics: true,
		},
		{
			name:               "comment with leading and trailing spaces",
			comment:            "   trimmed comment   ",
			wantNil:            false,
			wantLoggingQuery:   "-- trimmed comment",
			wantMonitoring:     "",
			wantSupportMetrics: true,
		},
		{
			name:    "empty comment returns nil",
			comment: "",
			wantNil: true,
		},
		{
			name:    "whitespace-only comment returns nil",
			comment: "     ",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Comment(tc.comment)
			if tc.wantNil {
				if m != nil {
					t.Errorf("Comment(%q) = %v, want nil", tc.comment, m)
				}
				return
			}

			if m == nil {
				t.Fatalf("Comment(%q) = nil, want non-nil", tc.comment)
			}

			if got := m.SupportMetrics(); got != tc.wantSupportMetrics {
				t.Errorf("SupportMetrics() = %v, want %v", got, tc.wantSupportMetrics)
			}

			if diff := cmp.Diff(tc.wantLoggingQuery, m.ToLoggingQuery()); diff != "" {
				t.Errorf("ToLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantMonitoring, m.ToMonitoringFilter()); diff != "" {
				t.Errorf("ToMonitoringFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWithCommentMatcher(t *testing.T) {
	testCases := []struct {
		name               string
		inner              LoggingMonitoringMatcher
		comment            string
		wantNil            bool
		wantLoggingQuery   string
		wantMonitoring     string
		wantSupportMetrics bool
	}{
		{
			name:               "ResourceLabel with comment",
			inner:              ResourceLabel("cluster_name", Exact("my-cluster")),
			comment:            "target cluster identity",
			wantNil:            false,
			wantLoggingQuery:   `resource.labels.cluster_name="my-cluster" -- target cluster identity`,
			wantMonitoring:     `resource.labels.cluster_name = "my-cluster"`,
			wantSupportMetrics: true,
		},
		{
			name:               "LogID with comment",
			inner:              LogID(Exact("events")),
			comment:            "k8s events only",
			wantNil:            false,
			wantLoggingQuery:   `LOG_ID("events") -- k8s events only`,
			wantMonitoring:     `metric.labels.log = "events"`,
			wantSupportMetrics: true,
		},
		{
			name:               "CustomFilter with comment",
			inner:              CustomFilter(`protoPayload.methodName="delete"`),
			comment:            "ignore deletion actions",
			wantNil:            false,
			wantLoggingQuery:   `protoPayload.methodName="delete" -- ignore deletion actions`,
			wantMonitoring:     "",
			wantSupportMetrics: false,
		},
		{
			name:               "nil inner matcher returns nil",
			inner:              nil,
			comment:            "should be nil",
			wantNil:            true,
			wantLoggingQuery:   "",
			wantMonitoring:     "",
			wantSupportMetrics: false,
		},
		{
			name:               "empty comment returns original inner matcher",
			inner:              ResourceLabel("project_id", Exact("test-project")),
			comment:            "",
			wantNil:            false,
			wantLoggingQuery:   `resource.labels.project_id="test-project"`,
			wantMonitoring:     `resource.labels.project_id = "test-project"`,
			wantSupportMetrics: true,
		},
		{
			name:               "whitespace-only comment returns original inner matcher",
			inner:              ResourceLabel("project_id", Exact("test-project")),
			comment:            "   ",
			wantNil:            false,
			wantLoggingQuery:   `resource.labels.project_id="test-project"`,
			wantMonitoring:     `resource.labels.project_id = "test-project"`,
			wantSupportMetrics: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := WithComment(tc.inner, tc.comment)
			if tc.wantNil {
				if m != nil {
					t.Errorf("WithComment() = %v, want nil", m)
				}
				return
			}

			if m == nil {
				t.Fatalf("WithComment() = nil, want non-nil")
			}

			if got := m.SupportMetrics(); got != tc.wantSupportMetrics {
				t.Errorf("SupportMetrics() = %v, want %v", got, tc.wantSupportMetrics)
			}

			if diff := cmp.Diff(tc.wantLoggingQuery, m.ToLoggingQuery()); diff != "" {
				t.Errorf("ToLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantMonitoring, m.ToMonitoringFilter()); diff != "" {
				t.Errorf("ToMonitoringFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStructuredLogQueryWithComments(t *testing.T) {
	sq := &StructuredLogQuery{
		ResourceTypes: []string{"k8s_cluster"},
		Filters: []LoggingMonitoringMatcher{
			WithComment(ResourceLabel("project_id", Exact("my-project")), "the main project"),
			Comment("following filter ignores noisy events"),
			WithComment(CustomFilter(`-sourceLocation.file="httplog.go"`), "scheduler spam"),
		},
	}

	wantLoggingQuery := `resource.type="k8s_cluster"
resource.labels.project_id="my-project" -- the main project
-- following filter ignores noisy events
-sourceLocation.file="httplog.go" -- scheduler spam`

	if diff := cmp.Diff(wantLoggingQuery, sq.GenerateCloudLoggingQuery()); diff != "" {
		t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
	}

	if sq.AllFiltersSupportMetrics() {
		t.Errorf("AllFiltersSupportMetrics() = true, want false due to CustomFilter")
	}

	wantMetricFilters := []string{
		`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "my-project"`,
	}

	if diff := cmp.Diff(wantMetricFilters, sq.GenerateMonitoringMetricFilters()); diff != "" {
		t.Errorf("GenerateMonitoringMetricFilters() mismatch (-want +got):\n%s", diff)
	}
}
