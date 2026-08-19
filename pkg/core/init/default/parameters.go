package defaultinit

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

var (
	// CommonParametersKey stores parsed CommonParameters.
	CommonParametersKey = typedmap.NewTypedKey[*parameters.CommonParameters]("khi.google.com/init/params/common")

	// ServerParametersKey stores parsed ServerParameters.
	ServerParametersKey = typedmap.NewTypedKey[*parameters.ServerParameters]("khi.google.com/init/params/server")

	// JobParametersKey stores parsed JobParameters.
	JobParametersKey = typedmap.NewTypedKey[*parameters.JobParameters]("khi.google.com/init/params/job")

	// AuthParametersKey stores parsed AuthParameters.
	AuthParametersKey = typedmap.NewTypedKey[*parameters.AuthParameters]("khi.google.com/init/params/auth")

	// DebugParametersKey stores parsed DebugParameters.
	DebugParametersKey = typedmap.NewTypedKey[*parameters.DebugParameters]("khi.google.com/init/params/debug")
)

const (
	// InitializerIDParameterStores registers parameter stores.
	InitializerIDParameterStores coreinit.InitializerID = "khi.default/parameter-stores"

	// InitializerIDParameterParse parses parameters and injects them into InitContext.
	InitializerIDParameterParse coreinit.InitializerID = "khi.default/parameter-parse"
)

// ParameterStoresInitializer registers the standard KHI parameter stores.
var ParameterStoresInitializer = &coreinit.Initializer{
	ID:     InitializerIDParameterStores,
	Before: []coreinit.InitializerID{InitializerIDParameterParse},
	Init: func(ctx *coreinit.InitContext) error {
		parameters.AddStore(parameters.Help)
		parameters.AddStore(parameters.Common)
		parameters.AddStore(parameters.Server)
		parameters.AddStore(parameters.Job)
		parameters.AddStore(parameters.Auth)
		parameters.AddStore(parameters.Debug)
		return nil
	},
}

// ParameterParseInitializer parses CLI flags and injects parameter stores into InitContext.
var ParameterParseInitializer = &coreinit.Initializer{
	ID: InitializerIDParameterParse,
	Dependencies: []coreinit.InitializerID{
		InitializerIDLogger,
	},
	Init: func(ctx *coreinit.InitContext) error {
		if err := parameters.Parse(); err != nil {
			return err
		}
		coreinit.Set(ctx, CommonParametersKey, parameters.Common)
		coreinit.Set(ctx, ServerParametersKey, parameters.Server)
		coreinit.Set(ctx, JobParametersKey, parameters.Job)
		coreinit.Set(ctx, AuthParametersKey, parameters.Auth)
		coreinit.Set(ctx, DebugParametersKey, parameters.Debug)
		return nil
	},
}

// RegisterParameterStore registers a parameter store that will be added before parameters.Parse is executed.
func RegisterParameterStore(id coreinit.InitializerID, store parameters.ParameterStore) {
	coreinit.RegisterInitializer(&coreinit.Initializer{
		ID:     id,
		Before: []coreinit.InitializerID{InitializerIDParameterParse},
		Init: func(ctx *coreinit.InitContext) error {
			parameters.AddStore(store)
			return nil
		},
	})
}

func init() {
	coreinit.RegisterInitializer(ParameterStoresInitializer)
	coreinit.RegisterInitializer(ParameterParseInitializer)
}
