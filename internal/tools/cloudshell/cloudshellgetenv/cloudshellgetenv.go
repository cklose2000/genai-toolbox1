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

package cloudshellgetenv

import (
	"context"
	"fmt"

	shell "cloud.google.com/go/shell/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudshellsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudshell"
	"github.com/googleapis/genai-toolbox/internal/tools"
	shellpb "google.golang.org/genproto/googleapis/cloud/shell/v1"
)

const kind string = "cloudshell-get-env"

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
	req := &shellpb.GetEnvironmentRequest{
		Name: t.EnvName,
	}

	env, err := t.Client.GetEnvironment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Build response with environment details
	result := map[string]any{
		"name":      env.Name,
		"id":        env.Id,
		"state":     env.State.String(),
		"docker_image": env.DockerImage,
	}

	// Add SSH connection info if available
	if env.SshHost != "" {
		result["ssh_host"] = env.SshHost
		result["ssh_port"] = env.SshPort
		result["ssh_username"] = env.SshUsername
	}

	// Add web host info if available
	if env.WebHost != "" {
		result["web_host"] = env.WebHost
	}

	return []any{result}, nil
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