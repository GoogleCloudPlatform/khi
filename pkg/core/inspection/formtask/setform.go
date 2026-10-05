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
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
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

// setFormTask holds the settings of a set form task defined with DefineSetForm.
type setFormTask[T any] struct {
	formTaskBase[T]
	defaultValue     SetFormDefaultValueGenerator
	validator        SetFormValidator
	optionsProvider  SetFormOptionsProvider
	hintGenerator    SetFormHintGenerator
	converter        SetFormValueConverter[T]
	allowCustomValue SetFormBoolProvider
	allowAddAll      SetFormBoolProvider
	allowRemoveAll   SetFormBoolProvider
}

// newSetFormTask creates a set form task with the default settings described in SetFormSpec.
func newSetFormTask[T any](id taskid.TaskImplementationID[T], priority int, label string, description string) *setFormTask[T] {
	return &setFormTask[T]{
		formTaskBase: newFormTaskBase(id, priority, label, description),
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

// run computes the form field metadata and returns the converted value of the current input.
func (b *setFormTask[T]) run(ctx context.Context) (T, error) {
	m := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	req := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInput)
	taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)
	globalSharedMap := khictx.MustGetValue(ctx, inspectioncore.GlobalSharedMap)

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

	if valueRaw, exist := req[b.id.ReferenceIDString()]; exist && valueRaw != nil {
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

	b.setupBaseFormField(&field.ParameterFormFieldBase)

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
	if validationErr != "" && taskMode == inspectioncore.TaskModeRun {
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
		if taskMode == inspectioncore.TaskModeRun {
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
}

// SetFormSpec holds the optional settings of a set form defined with DefineSetForm.
// Callbacks may read inputs declared on the Binder passed to the bind function.
// A nil callback or a zero value uses the default.
type SetFormSpec[T any] struct {
	// AllowCustomValue reports whether users can input custom values. Defaults to false.
	AllowCustomValue SetFormBoolProvider
	// AllowAddAll reports whether the "Add All" option is enabled. Defaults to true.
	AllowAddAll SetFormBoolProvider
	// AllowRemoveAll reports whether the "Remove All" option is enabled. Defaults to true.
	AllowRemoveAll SetFormBoolProvider
	// DefaultValue generates the default value of the set form. Defaults to nil.
	DefaultValue SetFormDefaultValueGenerator
	// Options provides the available options for the set form. Defaults to an empty slice.
	Options SetFormOptionsProvider
	// Validator validates the input string slice. Defaults to a validator that accepts any input.
	Validator SetFormValidator
	// Converter converts the input string slice to T. Defaults to a conversion that only works when T is []string.
	Converter SetFormValueConverter[T]
	// Hint generates a hint message for the form field. Defaults to no hint.
	Hint SetFormHintGenerator
}

func (b *setFormTask[T]) applySpec(spec SetFormSpec[T]) {
	if spec.AllowCustomValue != nil {
		b.allowCustomValue = spec.AllowCustomValue
	}
	if spec.AllowAddAll != nil {
		b.allowAddAll = spec.AllowAddAll
	}
	if spec.AllowRemoveAll != nil {
		b.allowRemoveAll = spec.AllowRemoveAll
	}
	if spec.DefaultValue != nil {
		b.defaultValue = spec.DefaultValue
	}
	if spec.Options != nil {
		b.optionsProvider = spec.Options
	}
	if spec.Validator != nil {
		b.validator = spec.Validator
	}
	if spec.Converter != nil {
		b.converter = spec.Converter
	}
	if spec.Hint != nil {
		b.hintGenerator = spec.Hint
	}
}

// DefineSetForm defines a set form task with a Binder.
// id, priority, label and description are required for every form. bind runs once, declares the inputs on the Binder,
// and returns the optional settings whose callbacks read those inputs. A form that reads no input returns its settings without declaring any.
func DefineSetForm[T any](id taskid.TaskImplementationID[T], priority int, label string, description string, bind func(b *coretask.Binder) SetFormSpec[T], labelOpts ...coretask.LabelOpt) coretask.DefinedTask[T] {
	form := newSetFormTask(id, priority, label, description)
	return coretask.Define(id, func(b *coretask.Binder) func(ctx context.Context) (T, error) {
		form.applySpec(bind(b))
		return form.run
	}, form.formLabelOpts(labelOpts)...)
}

// ConstantBool returns a SetFormBoolProvider that always returns value.
func ConstantBool(value bool) SetFormBoolProvider {
	return func(ctx context.Context) (bool, error) {
		return value, nil
	}
}

// ConstantOptions returns a SetFormOptionsProvider that always returns options.
func ConstantOptions(options ...inspectionmetadata.SetParameterFormFieldOptionItem) SetFormOptionsProvider {
	return func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		return options, nil
	}
}

// PreviousOrConstantDefaultValue returns a SetFormDefaultValueGenerator that returns the previous values when they exist, otherwise values.
func PreviousOrConstantDefaultValue(values []string) SetFormDefaultValueGenerator {
	return func(ctx context.Context, previousValues []string) ([]string, error) {
		if len(previousValues) > 0 {
			return previousValues, nil
		}
		return values, nil
	}
}
