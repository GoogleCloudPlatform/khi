package model

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type LogTypeDocumentModel struct {
	LogTypes []LogTypeDocumentElement
}

type LogTypeDocumentElement struct {
	ID        string
	Name      string
	ColorCode string
}

func GetLogTypeDocumentModel() LogTypeDocumentModel {
	logTypes := []LogTypeDocumentElement{}
	for i := 1; i < enum.EnumLogTypeCount; i++ {
		logType := enum.LogTypes[enum.LogType(i)]
		logTypes = append(logTypes, LogTypeDocumentElement{
			ID:        logType.EnumKeyName,
			Name:      logType.Label,
			ColorCode: strings.TrimLeft(logType.LabelBackgroundColor, "#"),
		})
	}
	return LogTypeDocumentModel{
		LogTypes: logTypes,
	}
}
