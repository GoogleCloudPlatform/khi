package config

import "github.com/GoogleCloudPlatform/khi/pkg/parameters"

// GetConfigResponse is the response type of /api/v2/config
type GetConfigResponse struct {
	// ViewerMode is a flag indicating if the server is a viewer mode and not accepting creating a new inspection request.
	ViewerMode bool
}

// NewGetConfigResponseFromParameters returns *GetConfigResponse created from given program parameters.
func NewGetConfigResponseFromParameters() *GetConfigResponse {
	isViewerMode := false
	if parameters.Server.ViewerMode != nil {
		isViewerMode = *parameters.Server.ViewerMode
	}
	return &GetConfigResponse{
		ViewerMode: isViewerMode,
	}
}
