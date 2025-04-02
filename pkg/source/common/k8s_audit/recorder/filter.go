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

package recorder

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
)

func AnyLogGroupFilter() LogGroupFilterFunc {
	return func(ctx context.Context, resourcePath string) bool {
		return true
	}
}

func ResourceKindLogGroupFilter(kindInSingular string) LogGroupFilterFunc {
	return func(ctx context.Context, resourcePath string) bool {
		pathSegments := strings.Split(resourcePath, "#")
		if len(pathSegments) == 4 {
			return kindInSingular == pathSegments[1]
		} else {
			return false
		}
	}
}

func SubresourceLogGroupFilter(subresource string) LogGroupFilterFunc {
	return func(ctx context.Context, resourcePath string) bool {
		pathSegments := strings.Split(resourcePath, "#")
		if len(pathSegments) == 5 {
			return subresource == pathSegments[4]
		} else {
			return false
		}
	}
}

func AnyLogFilter() LogFilterFunc {
	return func(ctx context.Context, l *types.AuditLogParserInput) bool {
		return true
	}
}

// OnlySucceedLogs returns a LogFilterFunc that only matches audit logs with non zero response code.
func OnlySucceedLogs() LogFilterFunc {
	return func(ctx context.Context, l *types.AuditLogParserInput) bool {
		return !l.IsErrorResponse
	}
}

func OnlyWithResourceBody() LogFilterFunc {
	return func(ctx context.Context, l *types.AuditLogParserInput) bool {
		return l.ResourceBodyReader != nil
	}
}

func OnlySpecificVerb(verb enum.RevisionVerb) LogFilterFunc {
	return func(ctx context.Context, l *types.AuditLogParserInput) bool {
		return l.Operation.Verb == verb
	}
}

func AndLogFilter(filters ...LogFilterFunc) LogFilterFunc {
	return func(ctx context.Context, l *types.AuditLogParserInput) bool {
		for _, filter := range filters {
			if !filter(ctx, l) {
				return false
			}
		}
		return true
	}
}
