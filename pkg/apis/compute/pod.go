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

package compute

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PodContainerCreateInput struct {
	// Container name
	Name string `json:"name"`
	// Image to use.
	Image string `json:"image"`
	// Command to execute (i.e., entrypoint for docker)
	Command []string `json:"command"`
	// Args for the Command (i.e. command for docker)
	Args []string `json:"args"`
	// Current working directory of the command.
	WorkingDir string `json:"working_dir"`
	// List of environment variable to set in the container.
	Envs []*ContainerKeyValue `json:"envs"`
}

type PodCreateInput struct {
	Containers []*PodContainerCreateInput `json:"containers"`
}

type ContainerDesc struct {
	Id string `json:"id"`
}
