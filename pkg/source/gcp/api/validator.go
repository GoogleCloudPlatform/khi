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

package api

import (
	"fmt"
	"strings"
)

var resourceNameRoots = []string{
	"projects",
	"organizations",
	"folders",
	"billingAccounts",
}

// ValidateResourceNameOnLogEntriesList validates the given resource name is valid or not as the argument of the item in resource names to call entries.list.
func ValidateResourceNameOnLogEntriesList(resourceName string) error {
	err := validateResourceNameBeginsWithResourceNameRoots(resourceName)
	if err != nil {
		return err
	}
	segments := strings.Split(resourceName, "/")
	switch len(segments) {
	case 2:
		if segments[1] == "" {
			return fmt.Errorf("resource name must have the id after the first slash. Please check this document: https://cloud.google.com/logging/docs/reference/v2/rest/v2/entries/list")
		}
		return nil
	case 8:
		return validateLogViewResourceName(segments)
	default:
		return fmt.Errorf("resource name must have 2 or 8 segments. Please check this document: https://cloud.google.com/logging/docs/reference/v2/rest/v2/entries/list")
	}
}

func validateResourceNameBeginsWithResourceNameRoots(resourceName string) error {
	for _, root := range resourceNameRoots {
		if strings.HasPrefix(resourceName, root) {
			return nil
		}
	}
	return fmt.Errorf("resource name must begin with one of the following: %v", resourceNameRoots)
}

func validateLogViewResourceName(resourceNameSegments []string) error {
	if resourceNameSegments[2] != "locations" {
		return fmt.Errorf("resource name must be in the format of `**/**/locations/**/buckets/**/views/**` but `locations` wasn't placed in the right place.")
	}
	if resourceNameSegments[4] != "buckets" {
		return fmt.Errorf("resource name must be in the format of `**/**/locations/**/buckets/**/views/**` but `buckets` wasn't placed in the right place.")
	}
	if resourceNameSegments[6] != "views" {
		return fmt.Errorf("resource name must be in the format of `**/**/locations/**/buckets/**/views/**` but `views` wasn't placed in the right place.")
	}
	return nil
}
