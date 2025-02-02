package config

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
)

func TestNewGetConfigResponseFromParameters(t *testing.T) {
	testCases := []struct {
		name       string
		viewerMode *bool
		want       *GetConfigResponse
	}{
		{
			name:       "viewer mode is nil",
			viewerMode: nil,
			want: &GetConfigResponse{
				ViewerMode: false,
			},
		},
		{
			name:       "viewer mode is true",
			viewerMode: testutil.P(true),
			want: &GetConfigResponse{
				ViewerMode: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parameters.Server.ViewerMode = tc.viewerMode
			got := NewGetConfigResponseFromParameters()
			if got.ViewerMode != tc.want.ViewerMode {
				t.Errorf("NewGetConfigResponseFromParameters() = %v, want %v", got, tc.want)
			}
		})

	}
}
