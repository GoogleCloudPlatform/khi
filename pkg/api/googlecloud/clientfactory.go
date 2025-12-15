// Copyright 2025 Google LLC
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

package googlecloud

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
	logging "cloud.google.com/go/logging/apiv2"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"google.golang.org/api/composer/v1"
	"google.golang.org/api/option"
)

// ClientFactoryContextModifiers defines a function type for modifying the context
// before creating a Google Cloud client.
type ClientFactoryContextModifiers = func(ctx context.Context, container ResourceContainer) (context.Context, error)

// ClientFactoryOptionsModifiers defines a function type for modifying the client options
// before creating a Google Cloud client.
type ClientFactoryOptionsModifiers = func(opts []option.ClientOption, container ResourceContainer) ([]option.ClientOption, error)

// ClientFactoryOption defines a function type for configuring a ClientFactory.
type ClientFactoryOption = func(s *ClientFactory) error

// ClientFactory generates a context used for generating the google cloud client.
// This type creates the instance of API clients centrally, it uses `cloud.google.com/go` package when the SDK supports the service,
// if not, it uses `google.golang.org/api` package for the service(e.g, Cloud Composer).
type ClientFactory struct {
	ClientOptions    []ClientFactoryOptionsModifiers
	ContextModifiers []ClientFactoryContextModifiers

	ContainerClusterManagerClientOptions []ClientFactoryOptionsModifiers
	LoggingClientOptions                 []ClientFactoryOptionsModifiers
	RegionsClientOptions                 []ClientFactoryOptionsModifiers
	ComposerServiceOptions               []ClientFactoryOptionsModifiers
	MonitoringMetricClientOptions        []ClientFactoryOptionsModifiers
}

// NewClientFactory creates a new ClientFactory with the given options.
// It applies each option to the factory and returns an error if any option fails.
func NewClientFactory(options ...ClientFactoryOption) (*ClientFactory, error) {
	var factory = &ClientFactory{}
	for _, opt := range options {
		err := opt(factory)
		if err != nil {
			return nil, err
		}
	}
	return factory, nil
}

// context applies all registered context modifiers to the given context for the given resource container.
func (s *ClientFactory) context(ctx context.Context, container ResourceContainer) (context.Context, error) {
	for _, modifier := range s.ContextModifiers {
		var err error
		ctx, err = modifier(ctx, container)
		if err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// options applies all registered client options modifiers to an initial set of client options
// for the given resource container. It returns the modified client options or an error if any modifier fails.
func (s *ClientFactory) options(container ResourceContainer, clientSpecificOptions []ClientFactoryOptionsModifiers, opts ...option.ClientOption) ([]option.ClientOption, error) {
	modifiers := append([]ClientFactoryOptionsModifiers{}, s.ClientOptions...)
	modifiers = append(modifiers, clientSpecificOptions...)
	var options []option.ClientOption
	for _, modifier := range modifiers {
		var err error
		options, err = modifier(options, container)
		if err != nil {
			return nil, err
		}
	}
	options = append(options, opts...)
	return options, nil
}

// prepareServiceInput returns the context and options needed for initializing clients.
func (s *ClientFactory) prepareServiceInput(ctx context.Context, c ResourceContainer, clientSpecificOptions []ClientFactoryOptionsModifiers, opts ...option.ClientOption) (context.Context, []option.ClientOption, error) {
	ctx, err := s.context(ctx, c)
	if err != nil {
		return nil, nil, err
	}
	options, err := s.options(c, clientSpecificOptions, opts...)

	return ctx, options, err
}

// ContainerClusterManagerClient returns the ClusterManagerClient of container.googleapis.com from given context and the resource container.
func (s *ClientFactory) ContainerClusterManagerClient(ctx context.Context, c ResourceContainer, opts ...option.ClientOption) (*container.ClusterManagerClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c, s.ContainerClusterManagerClientOptions, opts...)
	if err != nil {
		return nil, err
	}

	return container.NewClusterManagerClient(ctx, opts...)
}

// LoggingClient returns the client for logging.googleapis.com from given context and the resource container.
func (s *ClientFactory) LoggingClient(ctx context.Context, c ResourceContainer, opts ...option.ClientOption) (*logging.Client, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c, s.LoggingClientOptions, opts...)
	if err != nil {
		return nil, err
	}
	return logging.NewClient(ctx, opts...)
}

// RegionsClient returns the client for listing GCE regions. https://cloud.google.com/compute/docs/reference/rest/v1#rest-resource:-v1.regions
func (s *ClientFactory) RegionsClient(ctx context.Context, c ResourceContainer, opts ...option.ClientOption) (*compute.RegionsClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c, s.RegionsClientOptions, opts...)
	if err != nil {
		return nil, err
	}

	return compute.NewRegionsRESTClient(ctx, opts...)
}

// ComposerService returns the client for composer.googleapis.com from given context and the resource container.
// Cloud Composer has no package defined by 'cloud.google.com/go', this method returns the low level API client from 'google.golang.org/api/composer/v1'
func (s *ClientFactory) ComposerService(ctx context.Context, c ResourceContainer, opts ...option.ClientOption) (*composer.Service, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c, s.ComposerServiceOptions, opts...)
	if err != nil {
		return nil, err
	}

	return composer.NewService(ctx, opts...)
}

// MonitoringMetricClient returns the client for monitoring.googleapis.com from given context and the resource container.
func (s *ClientFactory) MonitoringMetricClient(ctx context.Context, c ResourceContainer, opts ...option.ClientOption) (*monitoring.MetricClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c, s.MonitoringMetricClientOptions, opts...)
	if err != nil {
		return nil, err
	}

	return monitoring.NewMetricClient(ctx, opts...)
}
