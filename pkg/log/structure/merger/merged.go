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

package merger

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredata"
)

type arrayMergeResultElementReference struct {
	From  structuredata.StructureData
	Index string
}

type arrayMergeResultSourceCache struct {
	References []*arrayMergeResultElementReference
}

type strategicMergePatchKeys struct {
	FieldKeys               []string
	StrategicMergeAttibutes []string
}

type strategicMergePatchArrayEleemnts struct {
	Elements      []structuredata.StructureData
	Replace       bool
	DeletedFields []structuredata.StructureData
}

type deleteFromPrimitiveListDirective struct {
	list []any
}

type setElementOrderDirective struct {
	list []any
}

type StrategicMergedStructureData struct {
	path                             string
	prev                             structuredata.StructureData
	patch                            structuredata.StructureData
	mergeKeyResolver                 *MergeConfigResolver
	deleteFromPrimitiveListDirective *deleteFromPrimitiveListDirective
	setElementOrderDirective         *setElementOrderDirective
	arrayMergeResultSourceCache      *arrayMergeResultSourceCache
}

func NewStrategicMergedStructureData(path string, prev structuredata.StructureData, patch structuredata.StructureData, mergeKeyResolver *MergeConfigResolver) *StrategicMergedStructureData {
	return &StrategicMergedStructureData{
		path:                             path,
		prev:                             prev,
		patch:                            patch,
		mergeKeyResolver:                 mergeKeyResolver,
		deleteFromPrimitiveListDirective: nil,
		arrayMergeResultSourceCache:      nil,
		setElementOrderDirective:         nil,
	}
}

// Keys implements structuredata.StructureData.
func (d *StrategicMergedStructureData) Keys() ([]string, error) {
	prevKeys, err := d.prev.Keys()
	if err != nil {
		return nil, err
	}
	patchKeys, err := d.patch.Keys()
	if err != nil {
		return nil, err
	}
	ty, err := d.Type()
	if err != nil {
		return nil, err
	}
	patchAttr, err := d.readScalarAsString(d.patch, "$patch")
	if patchAttr == "delete" {
		return []string{""}, nil
	}

	if ty == structuredata.StructuredTypeMap {
		keydiff := NewKeyDiff(prevKeys, patchKeys)
		strategicPatchKeys := splitPatchKeysByFieldsOrMergeAttributes(keydiff.OnlyInPatch)
		if err == nil {
			if patchAttr == "replace" {
				return append(keydiff.Both, strategicPatchKeys.FieldKeys...), nil
			}
		}
		result := []string{}
		result = append(result, prevKeys...)
		result = append(result, strategicPatchKeys.FieldKeys...)
		retainKeysAttr, err := d.readStringArray(d.patch, "$retainKeys")
		if err == nil {
			retainKeysInMap := map[string]struct{}{}
			for _, key := range retainKeysAttr {
				retainKeysInMap[key] = struct{}{}
			}
			retainResult := []string{}
			for _, key := range result {
				if _, in := retainKeysInMap[key]; in {
					retainResult = append(retainResult, key)
				}
			}
			return retainResult, nil
		}
		return result, nil
	} else if ty == structuredata.StructuredTypeArray {
		if d.arrayMergeResultSourceCache == nil {
			err := d.buildArrayMergeResultSourceCache()
			if err != nil {
				return nil, err
			}
		}
		return stringSequence(len(d.arrayMergeResultSourceCache.References)), nil

	} else {
		return []string{""}, nil
	}
}

// Type implements structuredata.StructureData.
func (d *StrategicMergedStructureData) Type() (structuredata.StructuredDataFieldType, error) {
	patchAttr, err := d.readScalarAsString(d.patch, "$patch")
	if err == nil && patchAttr == "delete" {
		return structuredata.StructuredTypeScalar, nil
	}
	return d.patch.Type()
}

// Value implements structuredata.StructureData.
func (d *StrategicMergedStructureData) Value(fieldName string) (any, error) {
	ty, err := d.Type()
	if err != nil {
		return nil, err
	}
	if ty == structuredata.StructuredTypeScalar {
		patchPolicy, err := d.readScalarAsString(d.patch, "$patch")
		if err == nil && patchPolicy == "delete" {
			return nil, nil
		}
		return d.patch.Value(fieldName)
	}
	if ty == structuredata.StructuredTypeMap {
		if fieldName == "" {
			return d, nil
		}
		patchPolicy, err := d.readScalarAsString(d.patch, "$patch")
		if err == nil && patchPolicy == "replace" {
			return d.patch.Value(fieldName)
		}
		retainKeysAttr, err := d.readStringArray(d.patch, "$retainKeys")
		if err == nil {
			kept := false
			for _, key := range retainKeysAttr {
				if key == fieldName {
					kept = true
					break
				}
			}
			if !kept {
				return fmt.Errorf("field not found:%s(patched by delete already)", fieldName), nil
			}
		}
		prevFound := false
		patchFound := false
		var patchValueStructure structuredata.StructureData
		var prevValueStructure structuredata.StructureData
		patchValue, err := d.patch.Value(fieldName)
		if err == nil {
			var convertible bool
			patchFound = true
			patchValueStructure, convertible = patchValue.(structuredata.StructureData)
			if !convertible {
				return nil, fmt.Errorf("unreachable. Map children was not the structure data")
			}
		}
		prevValue, err := d.prev.Value(fieldName)
		if err == nil {
			var convertible bool
			prevValueStructure, convertible = prevValue.(structuredata.StructureData)
			if !convertible {
				return nil, fmt.Errorf("unreachable. Map children was not the structure data")
			}
			prevValueType, err := prevValueStructure.Type()
			if err != nil {
				return nil, err
			}
			// Ignore if the previous value is nil
			if prevValueType == structuredata.StructuredTypeScalar {
				prevActualValue, err := prevValueStructure.Value("")
				if err != nil {
					return nil, err
				}
				if prevActualValue != nil {
					prevFound = true
				} else if !patchFound {
					return prevValueStructure, nil
				}
			} else {
				prevFound = true
			}
		}
		if prevFound && patchFound {
			result := NewStrategicMergedStructureData(fmt.Sprintf("%s.%s", d.path, fieldName), prevValueStructure, patchValueStructure, d.mergeKeyResolver)
			deleteFromPrimitiveList, err := d.readArray(d.patch, fmt.Sprintf("$deleteFromPrimitiveList/%s", fieldName))
			if err == nil {
				result.deleteFromPrimitiveListDirective = &deleteFromPrimitiveListDirective{
					list: deleteFromPrimitiveList,
				}
			}
			setOrderElementList, err := d.readArray(d.patch, fmt.Sprintf("$setElementOrder/%s", fieldName))
			if err == nil {
				result.setElementOrderDirective = &setElementOrderDirective{
					list: setOrderElementList,
				}
			}
			return result, nil
		} else if prevFound {
			result := NewStrategicMergedStructureData(fmt.Sprintf("%s.%s", d.path, fieldName), prevValueStructure, prevValueStructure, d.mergeKeyResolver)
			deleteFromPrimitiveList, err := d.readArray(d.patch, fmt.Sprintf("$deleteFromPrimitiveList/%s", fieldName))
			if err == nil {
				result.deleteFromPrimitiveListDirective = &deleteFromPrimitiveListDirective{
					list: deleteFromPrimitiveList,
				}
			}
			setOrderElementList, err := d.readArray(d.patch, fmt.Sprintf("$setElementOrder/%s", fieldName))
			if err == nil {
				result.setElementOrderDirective = &setElementOrderDirective{
					list: setOrderElementList,
				}
			}
			return result, nil
		} else if patchFound {
			result := NewStrategicMergedStructureData(fmt.Sprintf("%s.%s", d.path, fieldName), patchValueStructure, patchValueStructure, d.mergeKeyResolver)
			deleteFromPrimitiveList, err := d.readArray(d.patch, fmt.Sprintf("$deleteFromPrimitiveList/%s", fieldName))
			if err == nil {
				result.deleteFromPrimitiveListDirective = &deleteFromPrimitiveListDirective{
					list: deleteFromPrimitiveList,
				}
			}
			setOrderElementList, err := d.readArray(d.patch, fmt.Sprintf("$setElementOrder/%s", fieldName))
			if err == nil {
				result.setElementOrderDirective = &setElementOrderDirective{
					list: setOrderElementList,
				}
			}
			return result, nil
		} else {
			return nil, fmt.Errorf("field not found:%s", fieldName)
		}
	}
	if ty == structuredata.StructuredTypeArray {
		if fieldName == "" {
			return d, nil
		}
		if d.arrayMergeResultSourceCache == nil {
			err := d.buildArrayMergeResultSourceCache()
			if err != nil {
				return nil, err
			}
		}
		index, err := strconv.Atoi(fieldName)
		if err != nil {
			return nil, err
		}
		if index < 0 || index >= len(d.arrayMergeResultSourceCache.References) {
			return nil, fmt.Errorf("index out of range")
		}
		ref := d.arrayMergeResultSourceCache.References[index]
		if ref.Index == "" {
			return ref.From, nil
		}
		return ref.From.Value(ref.Index)
	}
	return nil, fmt.Errorf("unsupported type to merge")
}

func (d *StrategicMergedStructureData) buildArrayMergeResultSourceCache() error {
	d.arrayMergeResultSourceCache = &arrayMergeResultSourceCache{
		References: make([]*arrayMergeResultElementReference, 0),
	}
	patchKeys, err := d.patch.Keys()
	if err != nil {
		return err
	}
	strategy := d.mergeKeyResolver.GetMergeArrayStrategy(d.path)
	if strategy == MergeStrategyReplace {
		for _, key := range patchKeys {
			d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
				From:  d.patch,
				Index: key,
			})
		}
	} else {
		mergeKey, err := d.mergeKeyResolver.GetMergeKey(d.path)
		if err != nil {
			return err
		}
		patchElements, err := d.splitPatchKeysByElementsOrMergeAttributesInArray(d.patch)
		if err != nil {
			return err
		}
		if patchElements.Replace {
			for _, elem := range patchElements.Elements {
				d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
					From:  elem,
					Index: "",
				})
			}
			return nil
		}
		prevFields := []structuredata.StructureData{}
		prevKeys, err := d.prev.Keys()
		if err != nil {
			return err
		}
		if len(prevKeys) == 1 && prevKeys[0] == "" {
			// The previous value is nil
		} else {
			for _, key := range prevKeys {
				vany, err := d.prev.Value(key)
				if err != nil {
					return err
				}
				vst, convertible := vany.(structuredata.StructureData)
				if !convertible {
					return fmt.Errorf("expected vany to be convertible to structuredata, but didn't")
				}
				prevFields = append(prevFields, vst)
			}
		}
		prevFieldsKeyOrder := []string{}
		prevFieldsMap := map[string]structuredata.StructureData{}
		for _, field := range prevFields {
			keyValue, err := d.readScalar(field, mergeKey)
			if err != nil {
				return fmt.Errorf("merge key %s not found in prev field map", mergeKey)
			}
			hash, err := toHash(keyValue)
			if err != nil {
				return err
			}
			prevFieldsKeyOrder = append(prevFieldsKeyOrder, hash)
			prevFieldsMap[hash] = field
		}
		patchFieldKeyOrder := []string{}
		patchFieldMap := map[string]structuredata.StructureData{}
		for _, field := range patchElements.Elements {
			keyValue, err := d.readScalar(field, mergeKey)
			if err != nil {
				return fmt.Errorf("merge key %s not found in patch field map", mergeKey)
			}
			hash, err := toHash(keyValue)
			if err != nil {
				return err
			}
			patchFieldKeyOrder = append(patchFieldKeyOrder, hash)
			patchFieldMap[hash] = field
		}
		deletedMap := map[string]struct{}{}
		if d.deleteFromPrimitiveListDirective != nil {
			for _, primitive := range d.deleteFromPrimitiveListDirective.list {
				hash, err := toHash(primitive)
				if err != nil {
					return err
				}
				deletedMap[hash] = struct{}{}
			}
		}
		for _, field := range patchElements.DeletedFields {
			key, err := d.readScalarAsString(field, mergeKey)
			if err != nil {
				return err
			}
			deletedMap[key] = struct{}{}
		}
		keydiff := NewKeyDiffForArrayMerge(prevFieldsKeyOrder, patchFieldKeyOrder)
		setElementOrderKeys := []string{}
		if d.setElementOrderDirective != nil {
			for _, orderKeyValue := range d.setElementOrderDirective.list {
				if structuredata, convertible := orderKeyValue.(structuredata.StructureData); convertible {
					orderKeyValue, err = d.readScalar(structuredata, mergeKey)
					if err != nil {
						return err
					}
				}
				hash, err := toHash(orderKeyValue)
				if err != nil {
					return err
				}
				setElementOrderKeys = append(setElementOrderKeys, hash)
			}
		}
		orderedKeys := reorderArrayKeysForMerge(prevFieldsKeyOrder, patchFieldKeyOrder, setElementOrderKeys)
		bothKeys := map[string]struct{}{}
		for _, key := range keydiff.Both {
			bothKeys[key] = struct{}{}
		}
		for _, key := range orderedKeys {
			if _, found := deletedMap[key]; found {
				continue
			}
			if _, found := bothKeys[key]; found {
				d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
					From:  NewStrategicMergedStructureData(d.path+"[]", prevFieldsMap[key], patchFieldMap[key], d.mergeKeyResolver),
					Index: "",
				})
				continue
			}
			if _, found := patchFieldMap[key]; found {
				d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
					From:  patchFieldMap[key],
					Index: "",
				})
				continue
			}
			if _, found := prevFieldsMap[key]; found {
				d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
					From:  prevFieldsMap[key],
					Index: "",
				})
				continue
			}
			complemented, err := structuredata.DataFromYaml(fmt.Sprintf("%s: %s", mergeKey, key))
			if err != nil {
				return err
			}
			d.arrayMergeResultSourceCache.References = append(d.arrayMergeResultSourceCache.References, &arrayMergeResultElementReference{
				From:  complemented,
				Index: "",
			})
		}
	}
	return nil
}

func (d *StrategicMergedStructureData) readScalar(st structuredata.StructureData, fieldName string) (any, error) {
	if fieldName == "" {
		return st.Value(fieldName)
	}
	attr, err := st.Value(fieldName)
	if err != nil {
		return "", err
	}
	attrSt, convertible := attr.(structuredata.StructureData)
	if !convertible {
		return "", fmt.Errorf("expected the data to be structure data but wasn't")
	}
	attrStAny, err := attrSt.Value("")
	if err != nil {
		return "", err
	}
	return attrStAny, nil
}

func (d *StrategicMergedStructureData) readScalarAsString(st structuredata.StructureData, fieldName string) (string, error) {
	fieldAny, err := d.readScalar(st, fieldName)
	if err != nil {
		return "", err
	}
	fieldString, convertible := fieldAny.(string)
	if !convertible {
		return "", fmt.Errorf("field %s can't convert into string", fieldName)
	}
	return fieldString, nil
}

func (d *StrategicMergedStructureData) readArray(st structuredata.StructureData, fieldName string) ([]any, error) {
	attr, err := st.Value(fieldName)
	if err != nil {
		return nil, err
	}
	attrSt, convertible := attr.(structuredata.StructureData)
	if !convertible {
		return nil, fmt.Errorf("expected the data to be structure data but wasn't")
	}
	ty, err := attrSt.Type()
	if err != nil {
		return nil, err
	}
	if ty != structuredata.StructuredTypeArray {
		return nil, fmt.Errorf("expected an array but %s was given", ty)
	}
	result := []any{}
	keys, err := attrSt.Keys()
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		r, err := d.readScalar(attrSt, key)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (d *StrategicMergedStructureData) readStringArray(st structuredata.StructureData, fieldName string) ([]string, error) {
	source, err := d.readArray(st, fieldName)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, val := range source {
		valInStr, convertible := val.(string)
		if !convertible {
			return nil, fmt.Errorf("the array element can't be converted to string")
		}
		result = append(result, valInStr)
	}
	return result, nil
}

var _ structuredata.StructureData = (*StrategicMergedStructureData)(nil)

func splitPatchKeysByFieldsOrMergeAttributes(patchKeys []string) *strategicMergePatchKeys {
	fieldKeys := []string{}
	attributes := []string{}
	for _, key := range patchKeys {
		if key == "$patch" || key == "$retainKeys" || strings.HasPrefix(key, "$setElementOrder/") || strings.HasPrefix(key, "$deleteFromPrimitiveList/") {
			attributes = append(attributes, key)
		} else {
			fieldKeys = append(fieldKeys, key)
		}
	}
	return &strategicMergePatchKeys{
		FieldKeys:               fieldKeys,
		StrategicMergeAttibutes: attributes,
	}
}

func (d *StrategicMergedStructureData) splitPatchKeysByElementsOrMergeAttributesInArray(arrayData structuredata.StructureData) (*strategicMergePatchArrayEleemnts, error) {
	elements := []structuredata.StructureData{}
	deletedFields := []structuredata.StructureData{}
	patchByReplace := false
	keys, err := arrayData.Keys()
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		scalarOrMapAny, err := arrayData.Value(key)
		if err != nil {
			return nil, err
		}
		scalarOrMap, convertible := scalarOrMapAny.(structuredata.StructureData)
		if !convertible {
			return nil, fmt.Errorf("failed to convert to structuredata.StructureData")
		}
		ty, err := scalarOrMap.Type()
		if err != nil {
			return nil, err
		}
		if ty != structuredata.StructuredTypeMap {
			elements = append(elements, scalarOrMap)
		} else {
			patchField, err := d.readScalarAsString(scalarOrMap, "$patch")
			if err == nil {
				if patchField == "replace" {
					patchByReplace = true
				} else if patchField == "delete" {
					deletedFields = append(deletedFields, scalarOrMap)
				}
				continue
			}
			elements = append(elements, scalarOrMap)
		}
	}
	return &strategicMergePatchArrayEleemnts{
		Elements:      elements,
		Replace:       patchByReplace,
		DeletedFields: deletedFields,
	}, nil
}

func removeSetElements(list []any, patch []any, remove []any) ([]any, error) {
	retained := map[string]struct{}{}
	for _, v := range list {
		vhash, err := toHash(v)
		if err != nil {
			return nil, err
		}
		retained[vhash] = struct{}{}
	}
	for _, v := range patch {
		vhash, err := toHash(v)
		if err != nil {
			return nil, err
		}
		retained[vhash] = struct{}{}
	}
	for _, v := range remove {
		vhash, err := toHash(v)
		if err != nil {
			return nil, err
		}
		delete(retained, vhash)
	}
	result := []any{}
	for _, v := range list {
		vhash, err := toHash(v)
		if err != nil {
			return nil, err
		}
		if _, found := retained[vhash]; found {
			result = append(result, v)
		}
	}
	for _, v := range patch {
		vhash, err := toHash(v)
		if err != nil {
			return nil, err
		}
		if _, found := retained[vhash]; found {
			result = append(result, v)
		}
	}
	return result, nil
}

func reorderArrayKeysForMerge(prev []string, patch []string, setElementOrderDirective []string) []string {
	parallelList := []string{}
	liveList := []string{}
	liveOnlyList := []string{}
	patchMap := toIndexMap(patch)
	prevMap := toIndexMap(prev)
	directiveMap := toIndexMap(setElementOrderDirective)
	for _, directiveElem := range setElementOrderDirective {
		if _, found := patchMap[directiveElem]; !found {
			if _, found := prevMap[directiveElem]; !found {
				// Assume the item was existing from the past
				liveList = append(liveOnlyList, directiveElem)
			}
		}
	}
	for _, elem := range prev {
		isLiveOnly := false
		if _, found := directiveMap[elem]; !found {
			if _, found := patchMap[elem]; !found {
				isLiveOnly = true
			}
		}
		if isLiveOnly {
			liveOnlyList = append(liveOnlyList, elem)
		} else {
			liveList = append(liveList, elem)
		}
	}
	parallelList = append(parallelList, liveList...)
	parallelList = append(parallelList, patch...)
	parallelList = append(parallelList, setElementOrderDirective...)
	slices.Sort(parallelList)
	parallelList = slices.Compact(parallelList)
	liveMap := toIndexMap(liveList)
	liveOnlyMap := toIndexMap(liveOnlyList)
	slices.SortStableFunc(parallelList, func(a, b string) int {
		if bothInMap(directiveMap, a, b) {
			return directiveMap[a] - directiveMap[b]
		}
		if bothInMap(patchMap, a, b) {
			return patchMap[a] - patchMap[b]
		}
		if bothInMap(liveMap, a, b) {
			return liveMap[a] - liveMap[b]
		}
		if _, found := liveMap[a]; found {
			return -1
		}
		return 1
	})
	slices.SortStableFunc(liveOnlyList, func(a, b string) int {
		if bothInMap(directiveMap, a, b) {
			return directiveMap[a] - directiveMap[b]
		}
		if bothInMap(liveOnlyMap, a, b) {
			return liveOnlyMap[a] - liveOnlyMap[b]
		}
		if _, found := directiveMap[a]; found {
			return -1
		}
		return 1
	})
	return append(liveOnlyList, parallelList...)
}

func toHash(v any) (string, error) {
	if vStr, convertible := v.(string); convertible {
		return vStr, nil
	}
	if vInt, convertible := v.(int); convertible {
		return strconv.Itoa(vInt), nil
	}
	if vBool, convertible := v.(bool); convertible {
		return strconv.FormatBool(vBool), nil
	}
	return "", fmt.Errorf("given type was not hashable")
}

func toIndexMap(arr []string) map[string]int {
	result := map[string]int{}
	for i, elem := range arr {
		result[elem] = i
	}
	return result
}

func bothInMap[T any](mapSet map[string]T, a string, b string) bool {
	_, foundA := mapSet[a]
	if !foundA {
		return false
	}
	_, foundB := mapSet[b]
	return foundB
}

func stringSequence(length int) []string {
	result := []string{}
	for i := 0; i < length; i++ {
		result = append(result, strconv.Itoa(i))
	}
	return result
}
