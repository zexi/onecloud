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

type ContainerMountPropagation string

const (
	// No mount propagation ("private" in Linux terminology).
	MountPropagation_PROPAGATION_PRIVATE ContainerMountPropagation = "private"
	// Mounts get propagated from the host to the container ("rslave" in Linux).
	MountPropagation_PROPAGATION_HOST_TO_CONTAINER ContainerMountPropagation = "rslave"
	// Mounts get propagated from the host to the container and from the
	// container to the host ("rshared" in Linux).
	MountPropagation_PROPAGATION_BIDIRECTIONAL ContainerMountPropagation = "rshared"
)

type ContainerMount struct {
	// Path of the mount within the container.
	ContainerPath string `json:"container_path,omitempty"`
	// Path of the mount on the host. If the hostPath doesn't exist, then runtimes
	// should report error. If the hostpath is a symbolic link, runtimes should
	// follow the symlink and mount the real destination to container.
	HostPath string `json:"host_path,omitempty"`
	// If set, the mount is read-only.
	Readonly bool `json:"readonly,omitempty"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `json:"selinux_relabel,omitempty"`
	// Requested propagation mode.
	Propagation ContainerMountPropagation `json:"propagation,omitempty"`
}

type ContainerSpec struct {
	apis.ContainerSpec
	Devices []*ContainerDevice `json:"devices"`
	Mounts  []*ContainerMount  `json:"mounts"`
}

type ContainerDevice struct {
	IsolatedDeviceId string `json:"isolated_device_id"`
	Type             string `json:"type"`
	Path             string `json:"path"`
	Addr             string `json:"addr"`
	ContainerPath    string `json:"container_path"`
	Permissions      string `json:"permissions"`

	DiskId string `json:"disk_id"`
}

type ContainerCreateInput struct {
	Name    string         `json:"name"`
	GuestId string         `json:"guest_id"`
	Spec    *ContainerSpec `json:"spec"`
}

type ContainerPullImageAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	ServerAddress string `json:"server_address,omitempty"`
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identity_token,omitempty"`
	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registry_token,omitempty"`
}

type ContainerPullImageInput struct {
	Image      string                        `json:"image"`
	PullPolicy apis.ImagePullPolicy          `json:"pull_policy"`
	Auth       *ContainerPullImageAuthConfig `json:"auth"`
}
