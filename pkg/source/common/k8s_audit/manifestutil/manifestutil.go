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

package manifestutil

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type DeletionStatus = int

const (
	DeletionStatusNonDefined DeletionStatus = 0
	DeletionStatusDeleting   DeletionStatus = 1
	DeletionStatusDeleted    DeletionStatus = 2
)

// ParseDeletionStatus returns the current deletion status and deletion time of this resource.
func ParseDeletionStatus(ctx context.Context, resourceBodyReader *structurev2.NodeReader, operation *model.KubernetesObjectOperation) DeletionStatus {
	gracefulSeconds := -1
	var deletionTime *time.Time = nil
	if resourceBodyReader != nil {
		gracefulSeconds = resourceBodyReader.ReadIntOrDefault("metadata.deletionGracePeriodSeconds", -1)
		deletionTimeInstance, err := resourceBodyReader.ReadTimestamp("metadata.deletionTimestamp")
		if err == nil && deletionTimeInstance.Sub(time.Time{}) > 0 {
			deletionTime = &deletionTimeInstance
		}
	}

	if gracefulSeconds == -1 { // When the graceful second field is not available, the deletion status is only read from the verb recorded on the audit log.
		if deletionTime != nil {
			return DeletionStatusDeleted
		} else {
			if operation.Verb == enum.RevisionVerbDelete {
				return DeletionStatusDeleted
			}
			return DeletionStatusNonDefined
		}
	} else {
		if gracefulSeconds == 0 {
			return DeletionStatusDeleted
		} else {
			return DeletionStatusDeleting
		}
	}
}

// ParseCreationTime returns the creation time from the resource body.
func ParseCreationTime(resourceBodyReader *structurev2.NodeReader, defaultTime time.Time) time.Time {
	if resourceBodyReader != nil {
		creationTimestamp, err := getCreationTimeFromManifest(resourceBodyReader)
		if err != nil {
			return defaultTime
		}
		return creationTimestamp
	}
	return defaultTime
}

func getCreationTimeFromManifest(resourceBodyReader *structurev2.NodeReader) (time.Time, error) {
	creationTimestamp, err := resourceBodyReader.ReadString("metadata.creationTimestamp")
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, creationTimestamp)
}
