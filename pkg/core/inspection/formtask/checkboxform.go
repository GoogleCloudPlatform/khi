package formtask

import (
	"context"
	"fmt"
	"strconv"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
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

// CheckboxFormTaskBuilder builds a DAG task that renders and processes a checkbox form field.
type CheckboxFormTaskBuilder struct {
	FormTaskBuilderBase[bool]
	defaultValue     CheckboxFormDefaultValueGenerator
	validator        CheckboxFormValidator
	readonlyProvider CheckboxFormReadonlyProvider
	hintGenerator    CheckboxFormHintGenerator
}

// NewCheckboxFormTaskBuilder constructs a new CheckboxFormTaskBuilder instance.
func NewCheckboxFormTaskBuilder(id taskid.TaskImplementationID[bool], priority int, fieldLabel string) *CheckboxFormTaskBuilder {
	return &CheckboxFormTaskBuilder{
		FormTaskBuilderBase: NewFormTaskBuilderBase(id, priority, fieldLabel),
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

// WithDependencies sets upstream task dependencies for this checkbox task.
func (b *CheckboxFormTaskBuilder) WithDependencies(dependencies []taskid.UntypedTaskReference) *CheckboxFormTaskBuilder {
	b.FormTaskBuilderBase.WithDependencies(dependencies)
	return b
}

// WithDescription sets the explanatory description for the checkbox field.
func (b *CheckboxFormTaskBuilder) WithDescription(description string) *CheckboxFormTaskBuilder {
	b.FormTaskBuilderBase.WithDescription(description)
	return b
}

// WithValidator sets the validation function for the checkbox value.
func (b *CheckboxFormTaskBuilder) WithValidator(validator CheckboxFormValidator) *CheckboxFormTaskBuilder {
	b.validator = validator
	return b
}

// WithDefaultValue sets a constant default checked state.
func (b *CheckboxFormTaskBuilder) WithDefaultValue(defValue bool) *CheckboxFormTaskBuilder {
	return b.WithDefaultValueFunc(func(ctx context.Context) (bool, error) {
		return defValue, nil
	})
}

// WithDefaultValueConstant sets a constant default checked state.
func (b *CheckboxFormTaskBuilder) WithDefaultValueConstant(defValue bool) *CheckboxFormTaskBuilder {
	return b.WithDefaultValue(defValue)
}

// WithDefaultValueFunc sets dynamic generator for the default checked state.
func (b *CheckboxFormTaskBuilder) WithDefaultValueFunc(defFunc CheckboxFormDefaultValueGenerator) *CheckboxFormTaskBuilder {
	b.defaultValue = defFunc
	return b
}

// WithReadonly sets a constant readonly state for the checkbox.
func (b *CheckboxFormTaskBuilder) WithReadonly(readonly bool) *CheckboxFormTaskBuilder {
	return b.WithReadonlyFunc(func(ctx context.Context) (bool, error) {
		return readonly, nil
	})
}

// WithReadonlyFunc sets dynamic provider for the readonly state.
func (b *CheckboxFormTaskBuilder) WithReadonlyFunc(readonlyFunc CheckboxFormReadonlyProvider) *CheckboxFormTaskBuilder {
	b.readonlyProvider = readonlyFunc
	return b
}

// WithHintFunc sets dynamic generator for hint message and hint type.
func (b *CheckboxFormTaskBuilder) WithHintFunc(hintFunc CheckboxFormHintGenerator) *CheckboxFormTaskBuilder {
	b.hintGenerator = hintFunc
	return b
}

// Build creates a DAG task instance from this builder definition.
func (b *CheckboxFormTaskBuilder) Build(labelOpts ...common_task.LabelOpt) common_task.Task[bool] {
	return common_task.NewTask(b.id, b.dependencies, func(ctx context.Context) (bool, error) {
		m := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		req := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)
		taskMode := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskMode)

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
					return false, fmt.Errorf("request parameter `%s` was not a valid boolean in task %s: %w", b.id, b.id, err)
				}
				currentValue = parsed
			default:
				return false, fmt.Errorf("request parameter `%s` was not given as boolean or boolean string in task %s", b.id, b.id)
			}
		}

		field.Type = inspectionmetadata.Checkbox
		field.HintType = inspectionmetadata.Info

		b.SetupBaseFormField(&field.ParameterFormFieldBase)

		validationErr, err := b.validator(ctx, currentValue)
		if err != nil {
			return false, fmt.Errorf("validator for task `%s` returned an unrecoverable error: %w", b.id, err)
		}
		if validationErr != "" {
			currentValue, err = b.defaultValue(ctx)
			if err != nil {
				return false, fmt.Errorf("default value generator for task `%s` returned an error: %w", b.id, err)
			}
		}
		if validationErr != "" && taskMode == inspectioncore_contract.TaskModeRun {
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
	}, append(labelOpts, inspectioncore_contract.NewFormTaskLabelOpt(
		b.label,
		b.description,
	))...)
}
