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
	POD_STATUS_CREATING_CONTAINER      = "creating_container"
	POD_STATUS_CREATE_CONTAINER_FAILED = "create_container_failed"
	POD_STATUS_DELETING_CONTAINER      = "deleting_container"
	POD_STATUS_DELETE_CONTAINER_FAILED = "delete_container_failed"

	CONTAINER_STATUS_PULLING_IMAGE      = "pulling_image"
	CONTAINER_STATUS_PULL_IMAGE_FAILED  = "pull_image_failed"
	CONTAINER_STATUS_PULLED_IMAGE       = "pulled_image"
	CONTAINER_STATUS_CREATING           = "creating"
	CONTAINER_STATUS_CREATE_FAILED      = "create_failed"
	CONTAINER_STATUS_STARTING           = "starting"
	CONTAINER_STATUS_START_FAILED       = "start_failed"
	CONTAINER_STATUS_STOPPING           = "stopping"
	CONTAINER_STATUS_STOP_FAILED        = "stop_failed"
	CONTAINER_STATUS_SYNC_STATUS        = "sync_status"
	CONTAINER_STATUS_SYNC_STATUS_FAILED = "sync_status_failed"
	CONTAINER_STATUS_UNKNOWN            = "unknown"
	CONTAINER_STATUS_CREATED            = "created"
	CONTAINER_STATUS_EXITED             = "exited"
	CONTAINER_STATUS_RUNNING            = "running"
	CONTAINER_STATUS_DELETING           = "deleting"
	CONTAINER_STATUS_DELETE_FAILED      = "delete_failed"
)

const (
	POD_METADATA_CRI_ID       = "cri_id"
	POD_METADATA_CRI_CONFIG   = "cri_config"
	CONTAINER_METADATA_CRI_ID = "cri_id"
)

type PodContainerCreateInput struct {
	// Container name
	Name string `json:"name"`
	ContainerSpec
}

type PodPortMappingProtocol string

const (
	PodPortMappingProtocolTCP  = "tcp"
	PodPortMappingProtocolUDP  = "udp"
	PodPortMappingProtocolSCTP = "sctp"
)

type PodPortMapping struct {
	Protocol      PodPortMappingProtocol `json:"protocol"`
	ContainerPort int32                  `json:"container_port"`
	HostPort      int32                  `json:"host_port"`
	HostIp        string                 `json:"host_ip"`
}

type PodVolume struct {
	Name string         `json:"name"`
	Disk *PodVolumeDisk `json:"disk"`
}

type PodVolumeDisk struct {
	DiskIndex int `json:"disk_index"`
}

type PodCreateInput struct {
	Containers   []*PodContainerCreateInput `json:"containers"`
	PortMappings []*PodPortMapping          `json:"port_mappings"`
	Volumes      []*PodVolume               `json:"volumes"`
}

type PodStartResponse struct {
	CRIId     string `json:"cri_id"`
	IsRunning bool   `json:"is_running"`
}

type ContainerSyncStatusResponse struct {
	Status string `json:"status"`
}

type ContainerDesc struct {
	Id string `json:"id"`
}

type ContainerDevice struct {
	IsolatedDeviceId string `json:"isolated_device_id"`
}

type ContainerVolumeMount struct {
	// This must match the Name of a Volume.
	Name          string `json:"name"`
	AsRawDevice   bool   `json:"as_raw_device"`
	RawDevicePath string `json:"raw_device_path"`
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	ReadOnly bool `json:"read_only"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mount_path"`
}

type ContainerSpec struct {
	apis.ContainerSpec
	// Mounts for the container.
	// Mounts []*ContainerMount `json:"mounts"`
	Devices      []*ContainerDevice      `json:"devices"`
	VolumeMounts []*ContainerVolumeMount `json:"volume_mounts"`
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

type ContainerListInput struct {
	apis.VirtualResourceListInput
}

type ContainerStopInput struct {
	Timeout int `json:"timeout"`
}
