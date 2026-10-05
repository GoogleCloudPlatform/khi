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

// TextFormValidator is a function to check if the given value is valid or not.
// Returns "" as the result when it has no error, otherwise the returned value is used as an error message on frontend.
// Returning an error as the 2nd returning value is only when the validator detects an unrecoverble error.
type TextFormValidator = func(ctx context.Context, value string) (string, error)

// TextFormDefaultValueGenerator is a function type to generate the default value.
type TextFormDefaultValueGenerator = func(ctx context.Context, previousValues []string) (string, error)

// TextFormReadonlyProvider is a function type to compute if the field is allowed edit or not.
type TextFormReadonlyProvider = func(ctx context.Context) (bool, error)

// TextFormSuggestionsProvider is a function to return the list of strings shown in the autocomplete.
// Return nil instead of empty string array means the autocomplete is disabled for the field.
type TextFormSuggestionsProvider = func(ctx context.Context, value string, previousValues []string) ([]string, error)

// TextFormValueConverter is a function type to convert the given string value to another type stored in the variable set.
type TextFormValueConverter[T any] = func(ctx context.Context, value string) (T, error)

// TextFormHintGenerator is a function type to generate a hint string
type TextFormHintGenerator = func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error)

// textFormTask holds the settings of a text form task defined with DefineTextForm.
type textFormTask[T any] struct {
	formTaskBase[T]
	defaultValue        TextFormDefaultValueGenerator
	validator           TextFormValidator
	readonlyProvider    TextFormReadonlyProvider
	suggestionsProvider TextFormSuggestionsProvider
	hintGenerator       TextFormHintGenerator
	converter           TextFormValueConverter[T]
	validatingTiming    inspectionmetadata.TextFormValidationTimingType
}

// newTextFormTask creates a text form task with the default settings described in TextFormSpec.
func newTextFormTask[T any](id taskid.TaskImplementationID[T], priority int, label string, description string) *textFormTask[T] {
	return &textFormTask[T]{
		formTaskBase: newFormTaskBase(id, priority, label, description),
		defaultValue: func(ctx context.Context, previousValues []string) (string, error) {
			return "", nil
		},
		validator: func(ctx context.Context, value string) (string, error) {
			return "", nil
		},
		readonlyProvider: func(ctx context.Context) (bool, error) {
			return false, nil
		},
		suggestionsProvider: func(ctx context.Context, value string, previousValues []string) ([]string, error) {
			return nil, nil
		},
		converter: func(ctx context.Context, value string) (T, error) {
			var anyValue any = value // This is needed for forcible cast from string to T.
			return anyValue.(T), nil
		},
		hintGenerator: func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
			return "", inspectionmetadata.Info, nil
		},
		validatingTiming: inspectionmetadata.Change,
	}
}

// run computes the form field metadata and returns the converted value of the current input.
func (b *textFormTask[T]) run(ctx context.Context) (T, error) {
	m := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	req := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInput)
	taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)
	globalSharedMap := khictx.MustGetValue(ctx, inspectioncore.GlobalSharedMap)

	previousValueStoreKey := typedmap.NewTypedKey[[]string](fmt.Sprintf("text-form-pv-%s", b.id))
	prevValue := typedmap.GetOrDefault(globalSharedMap, previousValueStoreKey, []string{})

	readonly, err := b.readonlyProvider(ctx)
	if err != nil {
		return *new(T), fmt.Errorf("allowEdit provider for task `%s` returned an error\n%v", b.id, err)
	}
	field := inspectionmetadata.TextParameterFormField{}
	field.Readonly = readonly
	field.ValidationTiming = b.validatingTiming

	// Compute the default value of the form
	var currentValue string
	defaultValue, err := b.defaultValue(ctx, prevValue)
	if err != nil {
		return *new(T), fmt.Errorf("default value generator for task `%s` returned an error\n%v", b.id, err)
	}
	field.Default = defaultValue
	currentValue = defaultValue
	if valueRaw, exist := req[b.id.ReferenceIDString()]; exist && !readonly {
		valueString, isString := valueRaw.(string)
		if !isString {
			return *new(T), fmt.Errorf("request parameter `%s` was not given in string in task %s", b.id, b.id)
		}
		currentValue = valueString
	}

	field.Type = inspectionmetadata.Text
	field.HintType = inspectionmetadata.Info

	b.setupBaseFormField(&field.ParameterFormFieldBase)

	suggestions, err := b.suggestionsProvider(ctx, currentValue, prevValue)
	if err != nil {
		return *new(T), fmt.Errorf("suggesion provider for task `%s` returned an error\n%v", b.id, err)
	}
	field.Suggestions = suggestions

	validationErr, err := b.validator(ctx, currentValue)
	if err != nil {
		return *new(T), fmt.Errorf("validator for task `%s` returned an unrecoverable error\n%v", b.id, err)
	}
	if validationErr != "" {
		// When the given string is invalid, it should be the default value.
		currentValue, err = b.defaultValue(ctx, prevValue)
		if err != nil {
			return *new(T), fmt.Errorf("default value generator for task `%s` returned an error\n%v", b.id, err)
		}
	}
	if validationErr != "" && taskMode == inspectioncore.TaskModeRun {
		return *new(T), fmt.Errorf("validator for task `%s` returned a validation error. But this task was executed as a Run mode not in DryRun. All validations must be resolved before running.\n%v", b.id, validationErr)
	}

	convertedValue, err := b.converter(ctx, currentValue)
	if err != nil {
		return *new(T), fmt.Errorf("failed to convert the value `%s` to the dedicated value in task %s\n%v", currentValue, b.id, err)
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
			newValueHistory := append([]string{currentValue}, prevValue...)
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

// ConstantSuggestions returns a suggestions provider that always returns the given suggestions.
func ConstantSuggestions(suggestions ...string) TextFormSuggestionsProvider {
	return func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		return suggestions, nil
	}
}

// PreviousOrDefaultValue returns a TextFormDefaultValueGenerator that returns the most recent previous value when one exists,
// or calls defaultValue to compute the initial default value.
func PreviousOrDefaultValue(defaultValue func(ctx context.Context) (string, error)) TextFormDefaultValueGenerator {
	return func(ctx context.Context, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		return defaultValue(ctx)
	}
}

// TextFormSpec holds the optional settings of a text form defined with DefineTextForm.
// Callbacks may read inputs declared on the Binder passed to the bind function.
// A nil callback or a zero value uses the default.
type TextFormSpec[T any] struct {
	// Readonly reports whether the field is read only. Defaults to false.
	Readonly TextFormReadonlyProvider
	// DefaultValue generates the default value of the text form. Defaults to empty string.
	DefaultValue TextFormDefaultValueGenerator
	// Suggestions provides suggestions for the text form. Defaults to nil.
	Suggestions TextFormSuggestionsProvider
	// Validator validates the input string. Defaults to a validator that accepts any string.
	Validator TextFormValidator
	// Converter converts the input string to T. Defaults to a conversion that only works when T is string.
	Converter TextFormValueConverter[T]
	// Hint generates a hint message for the form field. Defaults to no hint.
	Hint TextFormHintGenerator
	// ValidationTiming specifies when form validation should occur. Defaults to inspectionmetadata.Change.
	ValidationTiming inspectionmetadata.TextFormValidationTimingType
}

func (b *textFormTask[T]) applySpec(spec TextFormSpec[T]) {
	if spec.Readonly != nil {
		b.readonlyProvider = spec.Readonly
	}
	if spec.DefaultValue != nil {
		b.defaultValue = spec.DefaultValue
	}
	if spec.Suggestions != nil {
		b.suggestionsProvider = spec.Suggestions
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
	if spec.ValidationTiming != "" {
		b.validatingTiming = spec.ValidationTiming
	}
}

// DefineTextForm defines a text form task with a Binder.
// id, priority, label and description are required for every form. bind runs once, declares the inputs on the Binder,
// and returns the optional settings whose callbacks read those inputs. A form that reads no input returns its settings without declaring any.
func DefineTextForm[T any](id taskid.TaskImplementationID[T], priority int, label string, description string, bind func(b *coretask.Binder) TextFormSpec[T], labelOpts ...coretask.LabelOpt) coretask.DefinedTask[T] {
	form := newTextFormTask(id, priority, label, description)
	return coretask.Define(id, func(b *coretask.Binder) func(ctx context.Context) (T, error) {
		form.applySpec(bind(b))
		return form.run
	}, form.formLabelOpts(labelOpts)...)
}
