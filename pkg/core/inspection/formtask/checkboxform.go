// Copyright 2026 Google LLC
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
	"strconv"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// CheckboxFormValidator validates whether the given boolean value is valid.
// Returns an empty string when valid, or an error message to display on the frontend.
// The second return value is used for unrecoverable errors during validation.
type CheckboxFormValidator = func(ctx context.Context, value bool) (string, error)

// CheckboxFormDefaultValueGenerator generates the default checked state for the checkbox.
type CheckboxFormDefaultValueGenerator = func(ctx context.Context) (bool, error)

// CheckboxFormReadonlyProvider determines whether the checkbox should be disabled/readonly.
type CheckboxFormReadonlyProvider = func(ctx context.Context) (bool, error)

// CheckboxFormHintGenerator generates a hint message and its severity type for the checkbox field.
type CheckboxFormHintGenerator = func(ctx context.Context, value bool) (string, inspectionmetadata.ParameterHintType, error)

// checkboxFormTask holds the settings of a checkbox form task defined with DefineCheckboxForm.
type checkboxFormTask struct {
	formTaskBase[bool]
	defaultValue     CheckboxFormDefaultValueGenerator
	validator        CheckboxFormValidator
	readonlyProvider CheckboxFormReadonlyProvider
	hintGenerator    CheckboxFormHintGenerator
}

// newCheckboxFormTask creates a checkbox form task with the default settings described in CheckboxFormSpec.
func newCheckboxFormTask(id taskid.TaskImplementationID[bool], priority int, label string, description string) *checkboxFormTask {
	return &checkboxFormTask{
		formTaskBase: newFormTaskBase(id, priority, label, description),
		defaultValue: func(ctx context.Context) (bool, error) {
			return false, nil
		},
		validator: func(ctx context.Context, value bool) (string, error) {
			return "", nil
		},
		readonlyProvider: func(ctx context.Context) (bool, error) {
			return false, nil
		},
		hintGenerator: func(ctx context.Context, value bool) (string, inspectionmetadata.ParameterHintType, error) {
			return "", inspectionmetadata.Info, nil
		},
	}
}

// run computes the form field metadata and returns the checked state of the current input.
func (b *checkboxFormTask) run(ctx context.Context) (bool, error) {
	m := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	req := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInput)
	taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)

	readonly, err := b.readonlyProvider(ctx)
	if err != nil {
		return false, fmt.Errorf("readonly provider for task `%s` returned an error: %w", b.id, err)
	}

	field := inspectionmetadata.CheckboxParameterFormField{}
	field.Readonly = readonly

	defaultValue, err := b.defaultValue(ctx)
	if err != nil {
		return false, fmt.Errorf("default value generator for task `%s` returned an error: %w", b.id, err)
	}
	field.Default = defaultValue
	currentValue := defaultValue

	if valueRaw, exist := req[b.id.ReferenceIDString()]; exist && !readonly {
		switch v := valueRaw.(type) {
		case bool:
			currentValue = v
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return false, fmt.Errorf("request parameter `%s` was not a valid boolean in task %s: %w", b.id.ReferenceIDString(), b.id, err)
			}
			currentValue = parsed
		default:
			return false, fmt.Errorf("request parameter `%s` was not given as boolean or boolean string in task %s", b.id.ReferenceIDString(), b.id)
		}
	}

	field.Type = inspectionmetadata.Checkbox
	field.HintType = inspectionmetadata.Info

	b.setupBaseFormField(&field.ParameterFormFieldBase)

	validationErr, err := b.validator(ctx, currentValue)
	if err != nil {
		return false, fmt.Errorf("validator for task `%s` returned an unrecoverable error: %w", b.id, err)
	}
	if validationErr != "" {
		currentValue = defaultValue
	}
	if validationErr != "" && taskMode == inspectioncore.TaskModeRun {
		return false, fmt.Errorf("validator for task `%s` returned a validation error. All validations must be resolved before running: %s", b.id, validationErr)
	}

	if validationErr != "" {
		field.HintType = inspectionmetadata.Error
		field.Hint = validationErr
	} else {
		hint, hintType, err := b.hintGenerator(ctx, currentValue)
		if err != nil {
			return false, fmt.Errorf("failed to generate a hint for task %s: %w", b.id, err)
		}
		if hint == "" {
			hintType = inspectionmetadata.None
		}
		field.Hint = hint
		field.HintType = hintType
	}

	formFields, found := typedmap.Get(m, inspectionmetadata.FormFieldSetMetadataKey)
	if !found {
		return false, fmt.Errorf("form field set was not found in the metadata set")
	}
	err = formFields.SetField(field)
	if err != nil {
		return false, fmt.Errorf("failed to configure the form metadata in task `%s`: %w", b.id, err)
	}
	return currentValue, nil
}

// CheckboxFormSpec holds the optional settings of a checkbox form defined with DefineCheckboxForm.
// Callbacks may read inputs declared on the Binder passed to the bind function.
// A nil callback uses the default.
type CheckboxFormSpec struct {
	// Readonly reports whether the field is read only. Defaults to false.
	Readonly CheckboxFormReadonlyProvider
	// DefaultValue generates the default checked state. Defaults to false.
	DefaultValue CheckboxFormDefaultValueGenerator
	// Validator validates the checked state. Defaults to a validator that accepts any value.
	Validator CheckboxFormValidator
	// Hint generates a hint message for the form field. Defaults to no hint.
	Hint CheckboxFormHintGenerator
}

func (b *checkboxFormTask) applySpec(spec CheckboxFormSpec) {
	if spec.Readonly != nil {
		b.readonlyProvider = spec.Readonly
	}
	if spec.DefaultValue != nil {
		b.defaultValue = spec.DefaultValue
	}
	if spec.Validator != nil {
		b.validator = spec.Validator
	}
	if spec.Hint != nil {
		b.hintGenerator = spec.Hint
	}
}

// DefineCheckboxForm defines a checkbox form task with a Binder.
// id, priority, label and description are required for every form. bind runs once, declares the inputs on the Binder,
// and returns the optional settings whose callbacks read those inputs. A form that reads no input returns its settings without declaring any.
func DefineCheckboxForm(id taskid.TaskImplementationID[bool], priority int, label string, description string, bind func(b *coretask.Binder) CheckboxFormSpec, labelOpts ...coretask.LabelOpt) coretask.DefinedTask[bool] {
	form := newCheckboxFormTask(id, priority, label, description)
	return coretask.Define(id, func(b *coretask.Binder) func(ctx context.Context) (bool, error) {
		form.applySpec(bind(b))
		return form.run
	}, form.formLabelOpts(labelOpts)...)
}
