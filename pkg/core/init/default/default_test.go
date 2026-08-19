package defaultinit

import (
	"context"
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

func TestDefaultInitializers_Resolution(t *testing.T) {
	parameters.ResetStore()
	defer parameters.ResetStore()

	engine := coreinit.NewEngine(context.Background())
	ctx := engine.Context()

	if err := engine.Init(); err != nil {
		t.Fatalf("Engine.Init() failed for default initializers: %v", err)
	}

	resolved, found := coreinit.Get(ctx, coreinit.ResolvedInitializersKey)
	if !found {
		t.Errorf("expected ResolvedInitializersKey to be present")
	} else if len(resolved) == 0 {
		t.Errorf("expected resolved initializers to be non-empty")
	}

	inspectionServer, found := coreinit.Get(ctx, InspectionTaskServerKey)
	if !found || inspectionServer == nil {
		t.Errorf("expected InspectionTaskServer to be created and set in context")
	}
}
