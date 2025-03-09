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

package queryutil

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Set filters are used for selecting namespaces, kinds or other sets for filtering purpose.
// Users will specify it as just string input field and it has specific syntax for selecting sets.
// It contains the following elements
// * `item` : just an additive element
// * `-item` : a subtractive element
// * `@alias_name` :  alias is predefined set of items. It will be expanded in the additive field.

type SetFilterAliasToItemsMap = map[string][]string

var validElementRegex = regexp.MustCompile(`^[@a-zA-Z0-9\-_]+$`)

type SetFilterParseResult struct {
	Additives       []string
	Subtractives    []string
	ValidationError string
	// The default mode is additive mode. The empty filter means none should be selected.
	// SubtractMode will match anything. The empty filter means anything should be selected, this field will be true when user specify @any in the field.
	SubtractMode bool
}

func ParseSetFilter(filter string, aliases SetFilterAliasToItemsMap, allowAny bool, allowSubtract bool, convertToLowerCase bool) (*SetFilterParseResult, error) {
	filterElements := strings.Split(filter, " ")
	for i := 0; i < len(filterElements); i++ {
		filterElements[i] = strings.TrimSpace(filterElements[i])
		if filterElements[i] != "" && !validElementRegex.Match([]byte(filterElements[i])) {
			return &SetFilterParseResult{ValidationError: "filter value must be whitespace splitted series of [a-zA-Z0-9\\-_]+"}, nil
		}
	}

	additivesMap := map[string]struct{}{}
	subtractivesMap := map[string]struct{}{}
	containsAny := false
	for i := 0; i < len(filterElements); i++ {
		filterElement := filterElements[i]
		if filterElement == "" {
			continue
		}
		negated := false
		if filterElement[0] == '-' {
			if !allowSubtract {
				return &SetFilterParseResult{
					ValidationError: "Subtract filter is not supported in this filter",
				}, nil
			}
			negated = true
			filterElement = filterElement[1:]
			if len(filterElement) == 0 {
				return &SetFilterParseResult{
					ValidationError: "Unexpected filter element `-`. Expecting item or alias after `-`.",
				}, nil
			}
		}
		if filterElement[0] == '@' {
			aliasName := filterElement[1:]
			if allowAny && aliasName == "any" {
				if negated {
					return &SetFilterParseResult{
						ValidationError: "Unsupported filter element `-@any`",
					}, nil
				}
				containsAny = true
				continue
			}
			if aliasMembers, found := aliases[aliasName]; found {
				for _, aliasMember := range aliasMembers {
					if negated {
						subtractivesMap[aliasMember] = struct{}{}
					} else {
						additivesMap[aliasMember] = struct{}{}
					}
				}
			} else {
				return &SetFilterParseResult{
					ValidationError: fmt.Sprintf("alias `%s` was not found", aliasName),
				}, nil
			}
		} else {
			if negated {
				subtractivesMap[filterElement] = struct{}{}
			} else {
				additivesMap[filterElement] = struct{}{}
			}
		}
	}

	result := &SetFilterParseResult{}
	result.Additives = []string{}
	result.Subtractives = []string{}
	result.SubtractMode = containsAny
	if containsAny {
		// If this was in subtract mode, we should evaluate subtract first then add next.
		for addOperand := range additivesMap {
			delete(subtractivesMap, addOperand)
		}
		for subtract := range subtractivesMap {
			result.Subtractives = append(result.Subtractives, subtract)
		}
	} else {
		// If this was in non subtract mode, we should evaluate add first then subtract next.
		for subtractOperand := range subtractivesMap {
			delete(additivesMap, subtractOperand)
		}
		for add := range additivesMap {
			result.Additives = append(result.Additives, add)
		}
	}
	slices.SortFunc(result.Additives, strings.Compare)
	slices.SortFunc(result.Subtractives, strings.Compare)
	if convertToLowerCase {
		result.Additives = ToLowerForStringArray(result.Additives)
		result.Subtractives = ToLowerForStringArray(result.Subtractives)
	}
	return result, nil
}
