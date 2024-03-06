// Copyright 2019 Yunion
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

package apis

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ContainerSpec struct {
	// Image to use.
	Image string `json:"image"`
	// Image pull policy
	ImagePullPolicy ImagePullPolicy `json:"image_pull_policy"`
	// Command to execute (i.e., entrypoint for docker)
	Command []string `json:"command"`
	// Args for the Command (i.e. command for docker)
	Args []string `json:"args"`
	// Current working directory of the command.
	WorkingDir string `json:"working_dir"`
	// List of environment variable to set in the container.
	Envs []*ContainerKeyValue `json:"envs"`
	// Enable lxcfs
	EnableLxcfs bool `json:"enable_lxcfs"`
}

type ImagePullPolicy string

const (
	ImagePullPolicyAlways       = "Always"
	ImagePullPolicyIfNotPresent = "IfNotPresent"
)
