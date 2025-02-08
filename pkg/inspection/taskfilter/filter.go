package taskfilter

// ContainsElement returns a function that represents a condition to filter only tasks that have the specified element in the specified label value.
func ContainsElement[T comparable](comparedWith T) func(taskLabelValueAny any) bool {
	return func(v any) bool {
		taskLabelValue := v.([]T)
		for _, element := range taskLabelValue {
			if element == comparedWith {
				return true
			}
		}
		return false
	}
}

// HasTrue is a function that represents a condition to filter only tasks with true value.
func HasTrue(taskLabelValueAny any) bool {
	return taskLabelValueAny.(bool)
}
