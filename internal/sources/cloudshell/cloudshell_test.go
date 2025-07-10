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

package cloudshell_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudshell"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCloudShell(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want []cloudshell.Config
	}{
		{
			desc: "basic cloud shell config",
			in: `sources:
  source1:
    kind: cloud-shell
    project: test-project
  source2:
    kind: cloud-shell
    project: test-project-2
    user: test-user`,
			want: []cloudshell.Config{
				{
					Name:    "source1",
					Kind:    "cloud-shell",
					Project: "test-project",
				},
				{
					Name:    "source2",
					Kind:    "cloud-shell",
					Project: "test-project-2",
					User:    "test-user",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var cfg server.Config
			if err := yaml.Unmarshal([]byte(tc.in), &cfg); err != nil {
				t.Fatal(err)
			}

			actual, err := testutils.SourceConfigs[cloudshell.Config](cfg)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, actual); diff != "" {
				t.Errorf("ParseFromYaml() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}