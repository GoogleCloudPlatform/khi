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

package googlecloud

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/googleapi"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mustStatusWithDetails(t *testing.T, code codes.Code, msg string, details ...*errdetails.ErrorInfo) error {
	t.Helper()
	st := status.New(code, msg)
	for _, d := range details {
		withDet, err := st.WithDetails(d)
		if err != nil {
			t.Fatalf("st.WithDetails() failed: %v", err)
		}
		st = withDet
	}
	return st.Err()
}

func mustRESTAPIError(t *testing.T, code int, msg, body string) error {
	t.Helper()
	ae, ok := apierror.FromError(&googleapi.Error{
		Code:    code,
		Message: msg,
		Body:    body,
	})
	if !ok {
		t.Fatalf("apierror.FromError() = false, want true")
	}
	return ae
}

func TestWrapPermissionDenied(t *testing.T) {
	t.Parallel()

	loggingDeniedWithDetails := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Permission 'logging.logEntries.list' denied on resource (or it may not exist).",
		&errdetails.ErrorInfo{
			Reason: "IAM_PERMISSION_DENIED",
			Domain: "iam.googleapis.com",
			Metadata: map[string]string{
				"permission": "logging.logEntries.list",
				"resource":   "projects/my-gcp-project",
			},
		},
	)

	loggingViewDenied := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Permission 'logging.views.access' denied on resource (or it may not exist).",
		&errdetails.ErrorInfo{
			Reason: "IAM_PERMISSION_DENIED",
			Domain: "iam.googleapis.com",
			Metadata: map[string]string{
				"permission": "logging.views.access",
				"resource":   "projects/my-gcp-project/locations/global/buckets/my-bucket/views/my-view",
			},
		},
	)

	quotaProjectDenied := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Caller does not have required permission to use project my-quota-project.",
		&errdetails.ErrorInfo{
			Reason: "USER_PROJECT_DENIED",
			Domain: "googleapis.com",
			Metadata: map[string]string{
				"permission": "serviceusage.services.use",
				"consumer":   "projects/my-quota-project",
			},
		},
	)

	serviceDisabled := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Cloud Logging API has not been used in project 123456789 before or it is disabled.",
		&errdetails.ErrorInfo{
			Reason: "SERVICE_DISABLED",
			Domain: "googleapis.com",
			Metadata: map[string]string{
				"service":  "logging.googleapis.com",
				"consumer": "projects/123456789",
			},
		},
	)

	scopeInsufficient := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Request had insufficient authentication scopes.",
		&errdetails.ErrorInfo{
			Reason: "ACCESS_TOKEN_SCOPE_INSUFFICIENT",
			Domain: "googleapis.com",
		},
	)

	billingDisabled := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"This API method requires billing to be enabled.",
		&errdetails.ErrorInfo{
			Reason: "BILLING_DISABLED",
			Domain: "googleapis.com",
			Metadata: map[string]string{
				"consumer": "projects/123456789",
			},
		},
	)

	restServiceDisabledAPIError := mustRESTAPIError(
		t,
		http.StatusForbidden,
		"Compute Engine API has not been used in project 123456789 before or it is disabled.",
		`{"error":{"code":403,"message":"Compute Engine API has not been used in project 123456789 before or it is disabled.","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SERVICE_DISABLED","domain":"googleapis.com","metadata":{"consumer":"projects/123456789","service":"compute.googleapis.com"}}]}}`,
	)

	restQuotaDeniedAPIError := mustRESTAPIError(
		t,
		http.StatusForbidden,
		"Caller does not have required permission to use project my-quota-project.",
		`{"error":{"code":403,"message":"Caller does not have required permission to use project my-quota-project.","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"USER_PROJECT_DENIED","domain":"googleapis.com","metadata":{"consumer":"projects/my-quota-project","permission":"serviceusage.services.use"}}]}}`,
	)

	unknownPermWithDetails := mustStatusWithDetails(
		t,
		codes.PermissionDenied,
		"Permission 'iam.serviceAccounts.getAccessToken' denied on resource.",
		&errdetails.ErrorInfo{
			Reason: "IAM_PERMISSION_DENIED",
			Domain: "iam.googleapis.com",
			Metadata: map[string]string{
				"permission": "iam.serviceAccounts.getAccessToken",
				"resource":   "projects/-/serviceAccounts/test-sa@my-gcp-project.iam.gserviceaccount.com",
			},
		},
	)

	plainErr := errors.New("connection reset by peer")
	unauthErr := status.Error(codes.Unauthenticated, "credentials expired")
	restForbiddenErr := &googleapi.Error{
		Code:    http.StatusForbidden,
		Message: "Required 'compute.regions.list' permission for 'projects/my-gcp-project'",
	}

	testCases := []struct {
		name              string
		err               error
		defaultResource   string
		defaultPermission string
		wantNil           bool
		wantUnchanged     bool
		wantMessage       string
		wantGRPCCode      codes.Code
	}{
		{
			name:              "nil error returns nil",
			err:               nil,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantNil:           true,
		},
		{
			name:              "non-API error is returned unchanged",
			err:               plainErr,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantUnchanged:     true,
		},
		{
			name:              "gRPC Unauthenticated error is returned unchanged",
			err:               unauthErr,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantUnchanged:     true,
		},
		{
			name:              "SERVICE_DISABLED error is returned unchanged",
			err:               serviceDisabled,
			defaultResource:   "projects/123456789",
			defaultPermission: "logging.logEntries.list",
			wantUnchanged:     true,
		},
		{
			name:              "ACCESS_TOKEN_SCOPE_INSUFFICIENT error is returned unchanged",
			err:               scopeInsufficient,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantUnchanged:     true,
		},
		{
			name:              "BILLING_DISABLED error is returned unchanged",
			err:               billingDisabled,
			defaultResource:   "projects/123456789",
			defaultPermission: "logging.logEntries.list",
			wantUnchanged:     true,
		},
		{
			name:              "GAPIC REST SERVICE_DISABLED APIError is returned unchanged",
			err:               restServiceDisabledAPIError,
			defaultResource:   "projects/123456789",
			defaultPermission: "compute.regions.list",
			wantUnchanged:     true,
		},
		{
			name:              "GAPIC REST USER_PROJECT_DENIED APIError preserves JSON ErrorInfo metadata",
			err:               restQuotaDeniedAPIError,
			defaultResource:   "projects/test-target-project",
			defaultPermission: "compute.regions.list",
			wantMessage:       "missing permission serviceusage.services.use on projects/my-quota-project; grant roles/serviceusage.serviceUsageConsumer to the active credentials: " + restQuotaDeniedAPIError.Error(),
		},
		{
			name:              "Cloud Logging gRPC PermissionDenied with ErrorInfo",
			err:               loggingDeniedWithDetails,
			defaultResource:   "projects/test-fallback-project",
			defaultPermission: "logging.logEntries.list",
			wantMessage:       "missing permission logging.logEntries.list on projects/my-gcp-project; grant roles/logging.viewer or roles/logging.privateLogViewer to the active credentials: " + loggingDeniedWithDetails.Error(),
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "Cloud Logging custom view logging.views.access recommends viewAccessor role",
			err:               loggingViewDenied,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantMessage:       "missing permission logging.views.access on projects/my-gcp-project/locations/global/buckets/my-bucket/views/my-view; grant roles/logging.viewAccessor or roles/logging.viewer or roles/logging.privateLogViewer to the active credentials: " + loggingViewDenied.Error(),
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "Cloud Monitoring gRPC PermissionDenied without ErrorInfo falls back to defaults",
			err:               status.Error(codes.PermissionDenied, "The caller does not have permission"),
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "monitoring.timeSeries.list",
			wantMessage:       "missing permission monitoring.timeSeries.list on projects/my-gcp-project; grant roles/monitoring.viewer to the active credentials: rpc error: code = PermissionDenied desc = The caller does not have permission",
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "Cloud Asset Inventory gRPC PermissionDenied",
			err:               status.Error(codes.PermissionDenied, "caller does not have permission"),
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "cloudasset.assets.searchAllResources",
			wantMessage:       "missing permission cloudasset.assets.searchAllResources on projects/my-gcp-project; grant roles/cloudasset.viewer to the active credentials: rpc error: code = PermissionDenied desc = caller does not have permission",
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "GKE container permission",
			err:               status.Error(codes.PermissionDenied, "caller does not have permission"),
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "container.clusters.list",
			wantMessage:       "missing permission container.clusters.list on projects/my-gcp-project; grant roles/container.viewer to the active credentials: rpc error: code = PermissionDenied desc = caller does not have permission",
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "REST HTTP 403 googleapi.Error extracts quoted permission from message",
			err:               restForbiddenErr,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "",
			wantMessage:       "missing permission compute.regions.list on projects/my-gcp-project; grant roles/compute.viewer to the active credentials: " + restForbiddenErr.Error(),
		},
		{
			name:              "Quota project USER_PROJECT_DENIED overrides default permission and resource",
			err:               quotaProjectDenied,
			defaultResource:   "projects/test-target-project",
			defaultPermission: "logging.logEntries.list",
			wantMessage:       "missing permission serviceusage.services.use on projects/my-quota-project; grant roles/serviceusage.serviceUsageConsumer to the active credentials: " + quotaProjectDenied.Error(),
			wantGRPCCode:      codes.PermissionDenied,
		},
		{
			name:              "Unknown permission in ErrorInfo omits specific role guess",
			err:               unknownPermWithDetails,
			defaultResource:   "projects/my-gcp-project",
			defaultPermission: "logging.logEntries.list",
			wantMessage:       "missing permission iam.serviceAccounts.getAccessToken on projects/-/serviceAccounts/test-sa@my-gcp-project.iam.gserviceaccount.com; grant an IAM role with this permission to the active credentials: " + unknownPermWithDetails.Error(),
			wantGRPCCode:      codes.PermissionDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := WrapPermissionDenied(tc.err, tc.defaultResource, tc.defaultPermission)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("WrapPermissionDenied() = %v, want nil", got)
				}
				return
			}
			if tc.wantUnchanged {
				if got != tc.err {
					t.Fatalf("WrapPermissionDenied() = %v, want unchanged %v", got, tc.err)
				}
				return
			}
			if got == nil {
				t.Fatalf("WrapPermissionDenied() = nil, want %q", tc.wantMessage)
			}
			if strings.Contains(tc.wantMessage, "\n") {
				if diff := cmp.Diff(tc.wantMessage, got.Error()); diff != "" {
					t.Errorf("WrapPermissionDenied().Error() mismatch (-want +got):\n%s", diff)
				}
			} else if got.Error() != tc.wantMessage {
				t.Errorf("WrapPermissionDenied().Error() = %q, want %q", got.Error(), tc.wantMessage)
			}
			if !errors.Is(got, tc.err) {
				t.Errorf("errors.Is(got, original) = false, want true")
			}
			if tc.wantGRPCCode != codes.OK && status.Code(got) != tc.wantGRPCCode {
				t.Errorf("status.Code(got) = %v, want %v", status.Code(got), tc.wantGRPCCode)
			}
			if _, ok := apierror.FromError(got); !ok {
				t.Errorf("apierror.FromError(got) = false, want true")
			}
			// Wrapping a second time must be idempotent.
			rewrapped := WrapPermissionDenied(got, "projects/test-other-project", "monitoring.timeSeries.list")
			if rewrapped != got {
				t.Errorf("second WrapPermissionDenied() = %v, want same error instance", rewrapped)
			}
		})
	}
}
