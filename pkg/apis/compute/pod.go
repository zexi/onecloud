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

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(new(ContainerSpec)), func() gotypes.ISerializable {
		return new(ContainerSpec)
	})
}

const (
	CONTAINER_STATUS_RUNNING = "running"
)

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PodContainerCreateInput struct {
	// Container name
	Name string `json:"name"`
	ContainerSpec
}

type PodCreateInput struct {
	Containers []*PodContainerCreateInput `json:"containers"`
}

type ContainerDesc struct {
	Id string `json:"id"`
}

type ContainerSpec struct {
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

func (c *ContainerSpec) String() string {
	return jsonutils.Marshal(c).String()
}

func (c *ContainerSpec) IsZero() bool {
	if reflect.DeepEqual(*c, ContainerSpec{}) {
		return true
	}
	return false
}

type ContainerCreateInput struct {
	apis.VirtualResourceCreateInput

	GuestId string        `json:"guest_id"`
	Spec    ContainerSpec `json:"spec"`
	// swagger:ignore
	SkipTask bool `json:"skip_task"`
}
