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

package cloudshell

import (
	"context"
	"fmt"

	shell "cloud.google.com/go/shell/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cloud-shell"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name    string `yaml:"name" validate:"required"`
	Kind    string `yaml:"kind" validate:"required"`
	Project string `yaml:"project" validate:"required"`
	User    string `yaml:"user"`  // Optional: specific user environment
}

func (cfg Config) SourceConfigKind() string {
	return SourceKind
}

func (cfg Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, cfg.Name)
	defer span.End()

	// Create Cloud Shell client
	client, err := shell.NewCloudShellClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Shell client: %w", err)
	}

	return &Source{
		Name:    cfg.Name,
		Kind:    cfg.Kind,
		Project: cfg.Project,
		User:    cfg.User,
		Client:  client,
		tracer:  tracer,
	}, nil
}

type Source struct {
	Name    string
	Kind    string
	Project string
	User    string
	Client  *shell.CloudShellClient
	tracer  trace.Tracer
}

// GetProject returns the configured project ID
func (s *Source) GetProject() string {
	return s.Project
}

// GetUser returns the configured user (if any)
func (s *Source) GetUser() string {
	return s.User
}

// CloudShellClient returns the Cloud Shell client
func (s *Source) CloudShellClient() *shell.CloudShellClient {
	return s.Client
}

// GetEnvironmentName returns the full environment resource name
func (s *Source) GetEnvironmentName() string {
	if s.User != "" {
		return fmt.Sprintf("users/%s/environments/default", s.User)
	}
	// Default to "me" for current authenticated user
	return "users/me/environments/default"
}

// validate interface
var _ sources.Source = &Source{}

func (s *Source) SourceKind() string {
	return s.Kind
}

func (s *Source) Cleanup() {
	if s.Client != nil {
		s.Client.Close()
	}
}