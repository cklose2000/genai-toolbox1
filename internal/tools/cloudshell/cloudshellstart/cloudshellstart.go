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

package cloudshellstart

import (
	"context"
	"fmt"
	"time"

	shell "cloud.google.com/go/shell/apiv1"
	shellpb "google.golang.org/genproto/googleapis/cloud/shell/v1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudshellsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudshell"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "cloudshell-start"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	CloudShellClient() *shell.CloudShellClient
	GetEnvironmentName() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudshellsrc.Source{}

var compatibleSources = [...]string{cloudshellsrc.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	// Build parameters
	parameters := tools.Parameters{}

	// Build manifests
	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// Create tool
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Client:       s.CloudShellClient(),
		EnvName:      s.GetEnvironmentName(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string
	Kind         string
	Parameters   tools.Parameters
	AuthRequired []string
	Client       *shell.CloudShellClient
	EnvName      string
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	// First, try to get the environment
	getReq := &shellpb.GetEnvironmentRequest{
		Name: t.EnvName,
	}

	env, err := t.Client.GetEnvironment(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Check if environment is already running
	if env.State == shellpb.Environment_RUNNING {
		return []any{map[string]any{
			"status": "already_running",
			"environment": map[string]any{
				"name":       env.Name,
				"id":         env.Id,
				"state":      env.State.String(),
				"ssh_host":   env.SshHost,
				"ssh_port":   env.SshPort,
				"ssh_username": env.SshUsername,
				"web_host":   env.WebHost,
			},
		}}, nil
	}

	// Start the environment
	startReq := &shellpb.StartEnvironmentRequest{
		Name: t.EnvName,
	}

	op, err := t.Client.StartEnvironment(ctx, startReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start environment: %w", err)
	}

	// Wait for the operation to complete (with timeout)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, err = op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for environment to start: %w", err)
	}
	
	// Get the environment details after starting
	getReq = &shellpb.GetEnvironmentRequest{
		Name: t.EnvName,
	}
	env, err = t.Client.GetEnvironment(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment after starting: %w", err)
	}

	// Return the started environment details
	return []any{map[string]any{
		"status": "started",
		"environment": map[string]any{
			"name":       env.Name,
			"id":         env.Id,
			"state":      env.State.String(),
			"ssh_host":   env.SshHost,
			"ssh_port":   env.SshPort,
			"ssh_username": env.SshUsername,
			"web_host":   env.WebHost,
		},
	}}, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}