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

package host

import "yunion.io/x/onecloud/pkg/apis"

type ContainerSpec struct {
	apis.ContainerSpec
	Devices []*ContainerDevice `json:"devices"`
}

type ContainerDevice struct {
	IsolatedDeviceId string `json:"isolated_device_id"`
	Type             string `json:"type"`
	Path             string `json:"path"`
	Addr             string `json:"addr"`
	ContainerPath    string `json:"container_path"`
}

type ContainerCreateInput struct {
	Name    string         `json:"name"`
	GuestId string         `json:"guest_id"`
	Spec    *ContainerSpec `json:"spec"`
}
