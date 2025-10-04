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

package googlecloudv2

import (
	"context"

	container "cloud.google.com/go/container/apiv1"
	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
	gkemulticloud "cloud.google.com/go/gkemulticloud/apiv1"
	logging "cloud.google.com/go/logging/apiv2"
	"google.golang.org/api/composer/v1"
	"google.golang.org/api/gkeonprem/v1"
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
	clientOptions    []ClientFactoryOptionsModifiers
	contextModifiers []ClientFactoryContextModifiers
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
	for _, modifier := range s.contextModifiers {
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
func (s *ClientFactory) options(container ResourceContainer) ([]option.ClientOption, error) {
	var options []option.ClientOption
	for _, modifier := range s.clientOptions {
		var err error
		options, err = modifier(options, container)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

// prepareServiceInput returns the context and options needed for initializing clients.
func (s *ClientFactory) prepareServiceInput(ctx context.Context, c ResourceContainer) (context.Context, []option.ClientOption, error) {
	ctx, err := s.context(ctx, c)
	if err != nil {
		return nil, nil, err
	}
	options, err := s.options(c)

	return ctx, options, err
}

// ContainerClusterManagerClient returns the ClusterManagerClient of container.googleapis.com from given context and the resource container.
func (s *ClientFactory) ContainerClusterManagerClient(ctx context.Context, c ResourceContainer) (*container.ClusterManagerClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return container.NewClusterManagerClient(ctx, opts...)
}

// GKEHubMembershipClient returns the MembershipClient of gkehub.googleapis.com from given context and the resource container.
func (s *ClientFactory) GKEHubMembershipClient(ctx context.Context, c ResourceContainer) (*gkehub.GkeHubMembershipClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return gkehub.NewGkeHubMembershipClient(ctx, opts...)
}

// GKEMultiCloudAWSClustersClient returns the AwsClusterClient of gkemulticloud.googleapis.com from given context and the resource container.
func (s *ClientFactory) GKEMultiCloudAWSClustersClient(ctx context.Context, c ResourceContainer) (*gkemulticloud.AwsClustersClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return gkemulticloud.NewAwsClustersClient(ctx, opts...)
}

// GKEMultiCloudAzureClustersClient returns the AzureClustersClient of gkemulticloud.googleapis.com from given context and the resource container.
func (s *ClientFactory) GKEMultiCloudAzureClustersClient(ctx context.Context, c ResourceContainer) (*gkemulticloud.AzureClustersClient, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return gkemulticloud.NewAzureClustersClient(ctx, opts...)
}

// LoggingClient returns the client for logging.googleapis.com from given context and the resource container.
func (s *ClientFactory) LoggingClient(ctx context.Context, c ResourceContainer) (*logging.Client, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}
	return logging.NewClient(ctx, opts...)
}

// ComposerService returns the client for composer.googleapis.com from given context and the resource container.
// Cloud Composer has no package defined by 'cloud.google.com/go', this method returns the low level API client from 'google.golang.org/api/composer/v1'
func (s *ClientFactory) ComposerService(ctx context.Context, c ResourceContainer) (*composer.Service, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return composer.NewService(ctx, opts...)
}

// GKEOnPremService returns the client for gkeonprem.googleapis.com from the given context and the resource cntainer.
// GKEOnPrem has no package defined by `cloud.google.com/go`, this method returns the low level API client from '"google.golang.org/api/gkeonprem/v1'.
func (s *ClientFactory) GKEOnPremService(ctx context.Context, c ResourceContainer) (*gkeonprem.Service, error) {
	ctx, opts, err := s.prepareServiceInput(ctx, c)
	if err != nil {
		return nil, err
	}

	return gkeonprem.NewService(ctx, opts...)
}
