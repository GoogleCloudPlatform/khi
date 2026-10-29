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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// deniedPermissionPattern extracts a quoted IAM permission name from Google Cloud gRPC and REST error messages
// when structured google.rpc.ErrorInfo metadata is absent.
var deniedPermissionPattern = regexp.MustCompile(`(?:[Pp]ermission|[Rr]equired) ['"]([a-z][a-z0-9]*\.[a-zA-Z0-9_.]+\.[a-zA-Z0-9_]+)['"]`)

type permissionDeniedError struct {
	msg string
	err error
}

func (e *permissionDeniedError) Error() string {
	return e.msg
}

func (e *permissionDeniedError) Unwrap() error {
	return e.err
}

// WrapPermissionDenied annotates Google Cloud permission errors with missing permissions and recommended IAM roles.
// It preserves the underlying error chain and returns non-permission or SERVICE_DISABLED errors unchanged.
func WrapPermissionDenied(err error, defaultResource, defaultPermission string) error {
	if err == nil {
		return nil
	}
	var existing *permissionDeniedError
	if errors.As(err, &existing) {
		return err
	}

	var ae *apierror.APIError
	if !errors.As(err, &ae) {
		var ok bool
		ae, ok = apierror.ParseError(err, false)
		if !ok {
			return err
		}
	}
	if status.Code(err) != codes.PermissionDenied && ae.HTTPCode() != http.StatusForbidden {
		return err
	}
	// Non-IAM 403/PermissionDenied reasons (disabled API, disabled billing, or insufficient OAuth scopes)
	// are not resolved by granting IAM roles.
	switch ae.Reason() {
	case "SERVICE_DISABLED", "BILLING_DISABLED", "ACCESS_TOKEN_SCOPE_INSUFFICIENT":
		return err
	}

	meta := ae.Metadata()
	permission := extractDeniedPermission(meta, err.Error(), defaultPermission)
	resource := extractDeniedResource(meta, defaultResource)
	roles := recommendedRolesForPermission(permission)

	subject := formatPermissionSubject(permission, resource)
	resolution := formatRoleResolution(permission, roles)

	return &permissionDeniedError{
		msg: fmt.Sprintf("%s%s: %s", subject, resolution, err.Error()),
		err: err,
	}
}

func extractDeniedPermission(meta map[string]string, message, fallback string) string {
	if p := meta["permission"]; p != "" {
		return p
	}
	if m := deniedPermissionPattern.FindStringSubmatch(message); len(m) == 2 {
		return m[1]
	}
	return fallback
}

func extractDeniedResource(meta map[string]string, fallback string) string {
	for _, key := range []string{"resource", "container", "consumer"} {
		if r := meta[key]; r != "" {
			return r
		}
	}
	return fallback
}

func recommendedRolesForPermission(permission string) []string {
	switch {
	case permission == "logging.views.access":
		return []string{"roles/logging.viewAccessor", "roles/logging.viewer", "roles/logging.privateLogViewer"}
	case strings.HasPrefix(permission, "logging."):
		return []string{"roles/logging.viewer", "roles/logging.privateLogViewer"}
	case strings.HasPrefix(permission, "monitoring."):
		return []string{"roles/monitoring.viewer"}
	case strings.HasPrefix(permission, "cloudasset."):
		return []string{"roles/cloudasset.viewer"}
	case strings.HasPrefix(permission, "compute."):
		return []string{"roles/compute.viewer"}
	case strings.HasPrefix(permission, "container."):
		return []string{"roles/container.viewer"}
	case strings.HasPrefix(permission, "serviceusage."):
		return []string{"roles/serviceusage.serviceUsageConsumer"}
	default:
		return nil
	}
}

func formatPermissionSubject(permission, resource string) string {
	switch {
	case permission != "" && resource != "":
		return fmt.Sprintf("missing permission %s on %s", permission, resource)
	case permission != "":
		return fmt.Sprintf("missing permission %s", permission)
	case resource != "":
		return fmt.Sprintf("permission denied on %s", resource)
	default:
		return "permission denied"
	}
}

func formatRoleResolution(permission string, roles []string) string {
	if len(roles) > 0 {
		return fmt.Sprintf("; grant %s to the active credentials", strings.Join(roles, " or "))
	}
	if permission != "" {
		return "; grant an IAM role with this permission to the active credentials"
	}
	return ""
}
