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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// CalloutFormGenerator generates a callout banner message and its severity type for a callout form field.
type CalloutFormGenerator = func(ctx context.Context) (string, inspectionmetadata.ParameterHintType, error)

// CalloutFormSpec configures a read-only callout banner task defined with DefineCalloutForm.
type CalloutFormSpec struct {
	// Callout returns the banner message and severity (Info, Warning, or Error).
	// Returning an empty message or inspectionmetadata.None omits the callout from the form.
	Callout CalloutFormGenerator
}

// calloutFormTask holds the settings of a read-only callout banner task defined with DefineCalloutForm.
type calloutFormTask struct {
	formTaskBase[struct{}]
	calloutGenerator CalloutFormGenerator
}

// DefineCalloutForm defines a read-only callout banner form task with a Binder.
//
// id, priority, label, and description configure the banner metadata. bind runs once, declares any upstream inputs
// on the Binder, and returns CalloutFormSpec whose Callout callback evaluates the banner message and severity.
func DefineCalloutForm(id taskid.TaskImplementationID[struct{}], priority int, label string, description string, bind func(b *coretask.Binder) CalloutFormSpec, labelOpts ...coretask.LabelOpt) coretask.Task[struct{}] {
	form := newCalloutFormTask(id, priority, label, description)
	return coretask.Define(id, func(b *coretask.Binder) func(ctx context.Context) (struct{}, error) {
		form.applySpec(bind(b))
		return form.run
	}, form.formLabelOpts(labelOpts)...)
}

// newCalloutFormTask creates a callout form task with the given base metadata.
func newCalloutFormTask(id taskid.TaskImplementationID[struct{}], priority int, label string, description string) *calloutFormTask {
	return &calloutFormTask{
		formTaskBase: newFormTaskBase(id, priority, label, description),
	}
}

func (b *calloutFormTask) applySpec(spec CalloutFormSpec) {
	if spec.Callout != nil {
		b.calloutGenerator = spec.Callout
	}
}

// run evaluates the callout generator and registers the callout form field metadata when active.
func (b *calloutFormTask) run(ctx context.Context) (struct{}, error) {
	if b.calloutGenerator == nil {
		return struct{}{}, nil
	}

	m := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)

	message, hintType, err := b.calloutGenerator(ctx)
	if err != nil {
		return struct{}{}, fmt.Errorf("callout generator for task `%s` returned an error: %w", b.id, err)
	}
	if message == "" || hintType == inspectionmetadata.None {
		return struct{}{}, nil
	}
	switch hintType {
	case "":
		hintType = inspectionmetadata.Info
	case inspectionmetadata.Info, inspectionmetadata.Warning, inspectionmetadata.Error:
	default:
		return struct{}{}, fmt.Errorf("callout generator for task `%s` returned unsupported hint type %q", b.id, hintType)
	}
	if hintType == inspectionmetadata.Error && taskMode == inspectioncore.TaskModeRun {
		return struct{}{}, fmt.Errorf("callout for task `%s` returned an error. All errors must be resolved before running: %s", b.id, message)
	}

	field := inspectionmetadata.CalloutParameterFormField{}
	b.setupBaseFormField(&field.ParameterFormFieldBase)
	field.Type = inspectionmetadata.Callout
	field.HintType = hintType
	field.Hint = message

	formFields, found := typedmap.Get(m, inspectionmetadata.FormFieldSetMetadataKey)
	if !found {
		return struct{}{}, fmt.Errorf("form field set was not found in the metadata set")
	}
	if err := formFields.SetField(field); err != nil {
		return struct{}{}, fmt.Errorf("failed to configure the form metadata in task `%s`: %w", b.id, err)
	}
	return struct{}{}, nil
}
