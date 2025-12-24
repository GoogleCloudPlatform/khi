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

package formtask

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// SetFormValidator is a function to check if the given value is valid or not.
type SetFormValidator = func(ctx context.Context, value []string) (string, error)

// SetFormDefaultValueGenerator is a function type to generate the default value.
type SetFormDefaultValueGenerator = func(ctx context.Context, previousValues []string) ([]string, error)

// SetFormOptionsProvider is a function to return the list of options.
type SetFormOptionsProvider = func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error)

// SetFormValueConverter is a function type to convert the given string slice value to another type stored in the variable set.
type SetFormValueConverter[T any] = func(ctx context.Context, value []string) (T, error)

// SetFormHintGenerator is a function type to generate a hint string
type SetFormHintGenerator = func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error)

// SetFormBoolProvider is a function type to provide boolean flags dynamically.
type SetFormBoolProvider = func(ctx context.Context) (bool, error)

// SetFormTaskBuilder is an utility to construct an instance of task for input form field.
type SetFormTaskBuilder[T any] struct {
	FormTaskBuilderBase[T]
	defaultValue     SetFormDefaultValueGenerator
	validator        SetFormValidator
	optionsProvider  SetFormOptionsProvider
	hintGenerator    SetFormHintGenerator
	converter        SetFormValueConverter[T]
	allowCustomValue SetFormBoolProvider
	allowAddAll      SetFormBoolProvider
	allowRemoveAll   SetFormBoolProvider
}

// NewSetFormTaskBuilder constructs an instance of SetFormTaskBuilder.
func NewSetFormTaskBuilder[T any](id taskid.TaskImplementationID[T], priority int, fieldLabel string) *SetFormTaskBuilder[T] {
	return &SetFormTaskBuilder[T]{
		FormTaskBuilderBase: NewFormTaskBuilderBase(id, priority, fieldLabel),
		defaultValue: func(ctx context.Context, previousValues []string) ([]string, error) {
			return nil, nil
		},
		validator: func(ctx context.Context, value []string) (string, error) {
			return "", nil
		},
		optionsProvider: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
			return []inspectionmetadata.SetParameterFormFieldOptionItem{}, nil
		},
		converter: func(ctx context.Context, value []string) (T, error) {
			var anyValue any = value
			if converted, convertible := anyValue.(T); convertible {
				return converted, nil
			}
			return *new(T), fmt.Errorf("value is not convertible to %T in the default converter. Did you forget to set the custom converter?", (*T)(nil))
		},
		hintGenerator: func(ctx context.Context, value []string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
			return "", inspectionmetadata.Info, nil
		},
		allowCustomValue: func(ctx context.Context) (bool, error) { return false, nil },
		allowAddAll:      func(ctx context.Context) (bool, error) { return true, nil },
		allowRemoveAll:   func(ctx context.Context) (bool, error) { return true, nil },
	}
}

func (b *SetFormTaskBuilder[T]) WithDependencies(dependencies []taskid.UntypedTaskReference) *SetFormTaskBuilder[T] {
	b.FormTaskBuilderBase.WithDependencies(dependencies)
	return b
}

func (b *SetFormTaskBuilder[T]) WithDescription(description string) *SetFormTaskBuilder[T] {
	b.FormTaskBuilderBase.WithDescription(description)
	return b
}

func (b *SetFormTaskBuilder[T]) WithValidator(validator SetFormValidator) *SetFormTaskBuilder[T] {
	b.validator = validator
	return b
}

func (b *SetFormTaskBuilder[T]) WithDefaultValueFunc(defFunc SetFormDefaultValueGenerator) *SetFormTaskBuilder[T] {
	b.defaultValue = defFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithDefaultValueConstant(defValue []string, preferPrevValue bool) *SetFormTaskBuilder[T] {
	return b.WithDefaultValueFunc(func(ctx context.Context, previousValues []string) ([]string, error) {
		if preferPrevValue {
			if len(previousValues) > 0 {
				return previousValues, nil
			}
		}
		return defValue, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithOptionsFunc(optionsFunc SetFormOptionsProvider) *SetFormTaskBuilder[T] {
	b.optionsProvider = optionsFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithOptionsConstant(options []inspectionmetadata.SetParameterFormFieldOptionItem) *SetFormTaskBuilder[T] {
	return b.WithOptionsFunc(func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		return options, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithOptionsSimple(options []string) *SetFormTaskBuilder[T] {
	return b.WithOptionsFunc(func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := make([]inspectionmetadata.SetParameterFormFieldOptionItem, len(options))
		for i, opt := range options {
			result[i] = inspectionmetadata.SetParameterFormFieldOptionItem{
				ID: opt,
			}
		}
		return result, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithAllowCustomValueFunc(allowFunc SetFormBoolProvider) *SetFormTaskBuilder[T] {
	b.allowCustomValue = allowFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithAllowCustomValue(allow bool) *SetFormTaskBuilder[T] {
	return b.WithAllowCustomValueFunc(func(ctx context.Context) (bool, error) {
		return allow, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithAllowAddAllFunc(allowFunc SetFormBoolProvider) *SetFormTaskBuilder[T] {
	b.allowAddAll = allowFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithAllowAddAll(allow bool) *SetFormTaskBuilder[T] {
	return b.WithAllowAddAllFunc(func(ctx context.Context) (bool, error) {
		return allow, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithAllowRemoveAllFunc(allowFunc SetFormBoolProvider) *SetFormTaskBuilder[T] {
	b.allowRemoveAll = allowFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithAllowRemoveAll(allow bool) *SetFormTaskBuilder[T] {
	return b.WithAllowRemoveAllFunc(func(ctx context.Context) (bool, error) {
		return allow, nil
	})
}

func (b *SetFormTaskBuilder[T]) WithHintFunc(hintFunc SetFormHintGenerator) *SetFormTaskBuilder[T] {
	b.hintGenerator = hintFunc
	return b
}

func (b *SetFormTaskBuilder[T]) WithConverter(converter SetFormValueConverter[T]) *SetFormTaskBuilder[T] {
	b.converter = converter
	return b
}

func (b *SetFormTaskBuilder[T]) Build(labelOpts ...common_task.LabelOpt) common_task.Task[T] {
	return common_task.NewTask(b.id, b.dependencies, func(ctx context.Context) (T, error) {
		m := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		req := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)
		taskMode := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskMode)
		globalSharedMap := khictx.MustGetValue(ctx, inspectioncore_contract.GlobalSharedMap)

		previousValueStoreKey := typedmap.NewTypedKey[[]string](fmt.Sprintf("set-form-pv-%s", b.id))
		prevValue := typedmap.GetOrDefault(globalSharedMap, previousValueStoreKey, []string{})

		allowCustomValue, err := b.allowCustomValue(ctx)
		if err != nil {
			return *new(T), fmt.Errorf("allowCustomValue provider for task `%s` returned an error\n%v", b.id, err)
		}
		allowAddAll, err := b.allowAddAll(ctx)
		if err != nil {
			return *new(T), fmt.Errorf("allowAddAll provider for task `%s` returned an error\n%v", b.id, err)
		}
		allowRemoveAll, err := b.allowRemoveAll(ctx)
		if err != nil {
			return *new(T), fmt.Errorf("allowRemoveAll provider for task `%s` returned an error\n%v", b.id, err)
		}

		field := inspectionmetadata.SetParameterFormField{}
		field.AllowCustomValue = allowCustomValue
		field.AllowAddAll = allowAddAll
		field.AllowRemoveAll = allowRemoveAll

		// Compute the default value
		var currentValue []string
		defaultValue, err := b.defaultValue(ctx, prevValue)
		if err != nil {
			return *new(T), fmt.Errorf("default value generator for task `%s` returned an error\n%v", b.id, err)
		}
		field.Default = defaultValue
		currentValue = defaultValue

		if valueRaw, exist := req[b.id.ReferenceIDString()]; exist {
			valueSlice, isSlice := valueRaw.([]interface{})
			if !isSlice {
				// Also try to handle string[] (though json unmarshal usually gives []interface{})
				if strSlice, ok := valueRaw.([]string); ok {
					currentValue = strSlice
				} else {
					return *new(T), fmt.Errorf("request parameter `%s` was not given in array in task %s", b.id, b.id)
				}
			} else {
				// Convert []interface{} to []string
				strs := make([]string, len(valueSlice))
				for i, v := range valueSlice {
					str, ok := v.(string)
					if !ok {
						return *new(T), fmt.Errorf("request parameter `%s` contains non-string value at index %d", b.id, i)
					}
					strs[i] = str
				}
				currentValue = strs
			}
		}

		field.Type = inspectionmetadata.Set
		field.HintType = inspectionmetadata.Info

		b.SetupBaseFormField(&field.ParameterFormFieldBase)

		options, err := b.optionsProvider(ctx, prevValue)
		if err != nil {
			return *new(T), fmt.Errorf("options provider for task `%s` returned an error\n%v", b.id, err)
		}
		field.Options = options

		validationErr, err := b.validator(ctx, currentValue)
		if err != nil {
			return *new(T), fmt.Errorf("validator for task `%s` returned an unrecoverable error\n%v", b.id, err)
		}
		if validationErr != "" {
			// When invalid, fallback to default
			currentValue, err = b.defaultValue(ctx, prevValue)
			if err != nil {
				return *new(T), fmt.Errorf("default value generator for task `%s` returned an error\n%v", b.id, err)
			}
		}
		if validationErr != "" && taskMode == inspectioncore_contract.TaskModeRun {
			return *new(T), fmt.Errorf("validator for task `%s` returned a validation error in Run mode. \n%v", b.id, validationErr)
		}

		convertedValue, err := b.converter(ctx, currentValue)
		if err != nil {
			return *new(T), fmt.Errorf("failed to convert the value `%v` to the dedicated value in task %s\n%v", currentValue, b.id, err)
		}

		if validationErr != "" {
			field.HintType = inspectionmetadata.Error
			field.Hint = validationErr
		} else {
			hint, hintType, err := b.hintGenerator(ctx, currentValue, convertedValue)
			if err != nil {
				return *new(T), fmt.Errorf("failed to generate a hint for task %s\n%v", b.id, err)
			}
			if hint == "" {
				hintType = inspectionmetadata.None
			}
			field.Hint = hint
			field.HintType = hintType
			if taskMode == inspectioncore_contract.TaskModeRun {
				newValueHistory := currentValue // Store current value as history
				typedmap.Set(globalSharedMap, previousValueStoreKey, newValueHistory)
			}
		}

		formFields, found := typedmap.Get(m, inspectionmetadata.FormFieldSetMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("form field set was not found in the metadata set")
		}
		err = formFields.SetField(field)
		if err != nil {
			return *new(T), fmt.Errorf("failed to configure the form metadata in task `%s`\n%v", b.id, err)
		}
		return convertedValue, nil
	}, append(labelOpts, inspectioncore_contract.NewFormTaskLabelOpt(
		b.label,
		b.description,
	))...)
}
