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
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typeddict"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ResourceNamesInput is a container for resource names used in log queries.
type ResourceNamesInput struct {
	resourceNames *typeddict.TypedDict[*QueryResourceNames]
}

// NewResourceNamesInput creates a new ResourceNamesInput.
func NewResourceNamesInput() *ResourceNamesInput {
	return &ResourceNamesInput{
		resourceNames: &typeddict.TypedDict[*QueryResourceNames]{},
	}
}

// QueryResourceNames holds the resource names for a specific query.
type QueryResourceNames struct {
	QueryID              string
	DefaultResourceNames []string
	CurrentResourceNames []string
}

// GetInputID returns the form input ID for the query.
func (q *QueryResourceNames) GetInputID() string {
	return fmt.Sprintf(GoogleCloudCommonTaskIDPrefix+"input-query-resource-names/%s", q.QueryID)
}

// UpdateDefaultResourceNamesForQuery updates the default resource names for a given query ID.
func (r *ResourceNamesInput) UpdateDefaultResourceNamesForQuery(queryID string, defaultResourceNames []string) {
	r.ensureQueryID(queryID)
	queryResourceNames := typeddict.GetOrDefault(r.resourceNames, queryID, &QueryResourceNames{})
	queryResourceNames.DefaultResourceNames = defaultResourceNames
}

// GetResourceNamesForQuery returns the resource names for a given query ID.
func (r *ResourceNamesInput) GetResourceNamesForQuery(ctx context.Context, queryID string) *QueryResourceNames {
	r.ensureQueryID(queryID)
	queryNames := typeddict.GetOrDefault(r.resourceNames, queryID, &QueryResourceNames{})

	currentResourceName := r.getResourceNamesFromInput(ctx, queryNames.GetInputID(), queryNames.DefaultResourceNames)

	if err := r.validateResourceNames(currentResourceName); err == nil {
		queryNames.CurrentResourceNames = currentResourceName
	} else {
		queryNames.CurrentResourceNames = queryNames.DefaultResourceNames
	}

	return queryNames
}

func (r *ResourceNamesInput) ensureQueryID(queryID string) {
	_, found := typeddict.Get(r.resourceNames, queryID)
	if !found {
		typeddict.Set(r.resourceNames, queryID, &QueryResourceNames{
			QueryID:              queryID,
			DefaultResourceNames: []string{},
		})
	}
}

func (r *ResourceNamesInput) getResourceNamesFromInput(ctx context.Context, inputID string, defaultResourceNames []string) []string {
	taskInput := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)

	inputAny, found := taskInput[inputID]
	if !found {
		return defaultResourceNames
	}
	if inputStr, ok := inputAny.(string); !ok {
		slog.WarnContext(ctx, "non string input was given for resource names", "inputID", inputID)
		return defaultResourceNames
	} else {
		resoruceNames := strings.Split(inputStr, " ")
		for i, name := range resoruceNames {
			resoruceNames[i] = strings.TrimSpace(name)
		}
		return resoruceNames
	}
}

func (r *ResourceNamesInput) validateResourceNames(resourceNames []string) error {
	for _, resourceName := range resourceNames {
		if err := googlecloud.ValidateResourceNameOnLogEntriesList(resourceName); err != nil {
			return err
		}
	}
	return nil
}
