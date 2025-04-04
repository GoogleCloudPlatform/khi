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

package form

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
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
// Return nil instead of emptry string array means the autocomplete is disabled for the field.
type TextFormSuggestionsProvider = func(ctx context.Context, value string, previousValues []string) ([]string, error)

// TextFormValueConverter is a function type to convert the given string value to another type stored in the variable set.
type TextFormValueConverter[T any] = func(ctx context.Context, value string) (T, error)

// TextFormHintGenerator is a function type to generate a hint string
type TextFormHintGenerator = func(ctx context.Context, value string, convertedValue any) (string, form_metadata.ParameterHintType, error)

// TextFormTaskBuilder is an utility to construct an instance of task for input form field.
// This will generate the task instance with `Build()` method call after chaining several configuration methods.
type TextFormTaskBuilder[T any] struct {
	FormTaskBuilderBase[T]
	defaultValue        TextFormDefaultValueGenerator
	validator           TextFormValidator
	readonlyProvider    TextFormReadonlyProvider
	suggestionsProvider TextFormSuggestionsProvider
	hintGenerator       TextFormHintGenerator
	converter           TextFormValueConverter[T]
}

// NewTextFormTaskBuilder constructs an instace of TextFormDefinitionBuilder.
// id,prioirity and label will be initialized with the value given in the argument. The other values are initialized with the following values.
// dependencies : Initialized with an empty string array indicating this task is not depending on anything.
// description: Initialized with an empty string.
// defaultValue: Initialized with a function to return empty string.
// validator: Initialized with a function to return empty string that indicates the validation is always passing.
// allowEditProvider: Initialized with a function to return true.
// suggestionsProvider: Initialized with a function to return nil.
// converter: Initialized with a function to return the given value. This means no conversion applied and treated as a string.
func NewTextFormTaskBuilder[T any](id taskid.TaskImplementationID[T], priority int, fieldLabel string) *TextFormTaskBuilder[T] {
	return &TextFormTaskBuilder[T]{
		FormTaskBuilderBase: NewFormTaskBuilderBase(id, priority, fieldLabel),
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
		hintGenerator: func(ctx context.Context, value string, convertedValue any) (string, form_metadata.ParameterHintType, error) {
			return "", form_metadata.Info, nil
		},
	}
}

func (b *TextFormTaskBuilder[T]) WithDependencies(dependencies []taskid.UntypedTaskReference) *TextFormTaskBuilder[T] {
	b.FormTaskBuilderBase.WithDependencies(dependencies)
	return b
}

func (b *TextFormTaskBuilder[T]) WithDescription(description string) *TextFormTaskBuilder[T] {
	b.FormTaskBuilderBase.WithDescription(description)
	return b
}

func (b *TextFormTaskBuilder[T]) WithValidator(validator TextFormValidator) *TextFormTaskBuilder[T] {
	b.validator = validator
	return b
}

func (b *TextFormTaskBuilder[T]) WithDefaultValueFunc(defFunc TextFormDefaultValueGenerator) *TextFormTaskBuilder[T] {
	b.defaultValue = defFunc
	return b
}

func (b *TextFormTaskBuilder[T]) WithDefaultValueConstant(defValue string, preferPrevValue bool) *TextFormTaskBuilder[T] {
	return b.WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if preferPrevValue {
			if len(previousValues) > 0 {
				return previousValues[0], nil
			}
		}
		return defValue, nil
	})
}

func (b *TextFormTaskBuilder[T]) WithReadonlyFunc(readonlyFunc TextFormReadonlyProvider) *TextFormTaskBuilder[T] {
	b.readonlyProvider = readonlyFunc
	return b
}

func (b *TextFormTaskBuilder[T]) WithSuggestionsFunc(suggestionsFunc TextFormSuggestionsProvider) *TextFormTaskBuilder[T] {
	b.suggestionsProvider = suggestionsFunc
	return b
}

func (b *TextFormTaskBuilder[T]) WithSuggestionsConstant(suggestions []string) *TextFormTaskBuilder[T] {
	return b.WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		return suggestions, nil
	})
}

func (b *TextFormTaskBuilder[T]) WithHintFunc(hintFunc TextFormHintGenerator) *TextFormTaskBuilder[T] {
	b.hintGenerator = hintFunc
	return b
}

func (b *TextFormTaskBuilder[T]) WithConverter(converter TextFormValueConverter[T]) *TextFormTaskBuilder[T] {
	b.converter = converter
	return b
}

func (b *TextFormTaskBuilder[T]) Build(labelOpts ...common_task.LabelOpt) common_task.Task[T] {
	return common_task.NewTask(b.id, b.dependencies, func(ctx context.Context) (T, error) {
		m := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
		req := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInput)
		taskMode := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskMode)
		globalSharedMap := khictx.MustGetValue(ctx, inspection_task_contextkey.GlobalSharedMap)

		previousValueStoreKey := typedmap.NewTypedKey[[]string](fmt.Sprintf("text-form-pv-%s", b.id))
		prevValue := typedmap.GetOrDefault(globalSharedMap, previousValueStoreKey, []string{})

		readonly, err := b.readonlyProvider(ctx)
		if err != nil {
			return *new(T), fmt.Errorf("allowEdit provider for task `%s` returned an error\n%v", b.id, err)
		}
		field := form_metadata.TextParameterFormField{}
		field.Readonly = readonly

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

		field.Type = form_metadata.Text
		field.HintType = form_metadata.Info

		b.SetupBaseFormField(&field.ParameterFormFieldBase)

		suggestions, err := b.suggestionsProvider(ctx, currentValue, prevValue)
		if err != nil {
			return *new(T), fmt.Errorf("suggesion provider for task `%s` returned an error\n%v", b.id, err)
		}
		field.Suggestions = suggestions

		validationErr, err := b.validator(ctx, currentValue)
		if err != nil {
			return *new(T), fmt.Errorf("validator for task `%s` returned an unrecovable error\n%v", b.id, err)
		}
		if validationErr != "" {
			// When the given string is invalid, it should be the default value.
			currentValue, err = b.defaultValue(ctx, prevValue)
			if err != nil {
				return *new(T), fmt.Errorf("default value generator for task `%s` returned an error\n%v", b.id, err)
			}
		}
		if validationErr != "" && taskMode == inspection_task_interface.TaskModeRun {
			return *new(T), fmt.Errorf("validator for task `%s` returned a validation error. But this task was executed as a Run mode not in DryRun. All validations must be resolved before running.\n%v", b.id, validationErr)
		}

		convertedValue, err := b.converter(ctx, currentValue)
		if err != nil {
			return *new(T), fmt.Errorf("failed to convert the value `%s` to the dedicated value in task %s\n%v", currentValue, b.id, err)
		}
		if validationErr != "" {
			field.HintType = form_metadata.Error
			field.Hint = validationErr
		} else {
			hint, hintType, err := b.hintGenerator(ctx, currentValue, convertedValue)
			if err != nil {
				return *new(T), fmt.Errorf("failed to generate a hint for task %s\n%v", b.id, err)
			}
			if hint == "" {
				hintType = form_metadata.None
			}
			field.Hint = hint
			field.HintType = hintType
			if taskMode == inspection_task_interface.TaskModeRun {
				newValueHistory := append([]string{currentValue}, prevValue...)
				typedmap.Set(globalSharedMap, previousValueStoreKey, newValueHistory)
			}
		}
		formFields, found := typedmap.Get(m, form_metadata.FormFieldSetMetadataKey)
		if !found {
			return *new(T), fmt.Errorf("form field set was not found in the metadata set")
		}
		err = formFields.SetField(field)
		if err != nil {
			return *new(T), fmt.Errorf("failed to configure the form metadata in task `%s`\n%v", b.id, err)
		}
		return convertedValue, nil
	}, append(labelOpts, label.NewFormTaskLabelOpt(
		b.label,
		b.description,
	))...)
}
