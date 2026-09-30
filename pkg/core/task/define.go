package coretask

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// DefinedTask is a task defined with Define. It exposes its typed input declarations to callers outside the task.
type DefinedTask[T any] interface {
	Task[T]
	// Inputs returns the inputs declared through the Binder in declaration order.
	Inputs() []InputSpec
}

type definedTaskImpl[T any] struct {
	*TaskImpl[T]
	inputs []InputSpec
}

var _ DefinedTask[any] = (*definedTaskImpl[any])(nil)

// Inputs implements DefinedTask.
func (t *definedTaskImpl[T]) Inputs() []InputSpec {
	return t.inputs
}

// Define constructs a task whose dependencies are declared through a Binder.
// bind runs once when Define is called. It must declare every input with Use, UseOptional, UseTag, or After,
// and return the run function that reads the inputs through the returned handles.
// Keep handles in local variables of bind so that the compiler reports inputs that are declared but never read.
func Define[T any](
	id taskid.TaskImplementationID[T],
	bind func(b *Binder) func(ctx context.Context) (T, error),
	labelOpts ...LabelOpt,
) DefinedTask[T] {
	verifyTaskID(id)
	b := newBinder(id)
	run := bind(b)
	b.sealed = true
	return &definedTaskImpl[T]{
		TaskImpl: NewTask(id, b.dependencies(), func(ctx context.Context) (T, error) {
			return run(withActiveDefinedTask(ctx, id))
		}, labelOpts...),
		inputs: b.specs,
	}
}
