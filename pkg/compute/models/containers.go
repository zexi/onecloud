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

package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var containerManager *SContainerManager

func GetContainerManager() *SContainerManager {
	if containerManager == nil {
		containerManager = &SContainerManager{
			SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
				SContainer{},
				"containers_tbl",
				"container",
				"containers"),
		}
		containerManager.SetVirtualObject(containerManager)
	}
	return containerManager
}

func init() {
	GetContainerManager()
}

type SContainerManager struct {
	db.SVirtualResourceBaseManager
}

type SContainer struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId string `width:"36" charset:"ascii" create:"required" list:"user" index:"true"`
	// Spec stores all container running options
	Spec *api.ContainerSpec `length:"long" create:"required" list:"user"`
}

func (m *SContainerManager) CreateOnPod(
	ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	pod *SGuest, data *api.PodContainerCreateInput) (*SContainer, error) {
	input := &api.ContainerCreateInput{
		GuestId:  pod.GetId(),
		Spec:     data.ContainerSpec,
		SkipTask: true,
	}
	input.Name = data.Name
	obj, err := db.DoCreate(m, ctx, userCred, nil, jsonutils.Marshal(input), ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "create container")
	}
	return obj.(*SContainer), nil
}

func (m *SContainerManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	guestId, _ := data.GetString("guest_id")
	return jsonutils.Marshal(map[string]string{"guest_id": guestId})
}

func (m *SContainerManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	guestId, _ := values.GetString("guest_id")
	if len(guestId) > 0 {
		q = q.Equals("guest_id", guestId)
	}
	return q
}

func (m *SContainerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ContainerListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (m *SContainerManager) GetContainersByPod(guestId string) ([]SContainer, error) {
	q := m.Query().Equals("guest_id", guestId)
	ctrs := make([]SContainer, 0)
	if err := db.FetchModelObjects(m, q, &ctrs); err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return ctrs, nil
}

func (m *SContainerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.ContainerCreateInput) (*api.ContainerCreateInput, error) {
	if input.GuestId == "" {
		return nil, httperrors.NewNotEmptyError("guest_id is required")
	}
	obj, err := GuestManager.FetchByIdOrName(userCred, input.GuestId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch guest by %s", input.GuestId)
	}
	pod := obj.(*SGuest)
	input.GuestId = pod.GetId()
	vols, err := m.GetPodVolumesByObject(ctx, userCred, pod)
	if err != nil {
		return nil, errors.Wrap(err, "GetPodVolumes by pod")
	}
	if err := m.ValidateSpec(ctx, userCred, &input.Spec, pod, vols); err != nil {
		return nil, errors.Wrap(err, "validate spec")
	}
	return input, nil
}

func (m *SContainerManager) ValidateSpecHostDevice(dev *api.ContainerHostDevice) error {
	if dev.HostPath == "" {
		return httperrors.NewNotEmptyError("host_path is empty")
	}
	if dev.ContainerPath == "" {
		return httperrors.NewNotEmptyError("container_path is empty")
	}
	if dev.Permissions == "" {
		return httperrors.NewNotEmptyError("permissions is empty")
	}
	for _, p := range strings.Split(dev.Permissions, "") {
		switch p {
		case "r", "w", "m":
		default:
			return httperrors.NewInputParameterError("wrong permission %s", p)
		}
	}
	return nil
}

func (m *SContainerManager) ValidateSpec(ctx context.Context, userCred mcclient.TokenCredential, spec *api.ContainerSpec, pod *SGuest, volumes []*api.PodVolume) error {
	for _, dev := range spec.Devices {
		if dev.IsolatedDeviceId != "" {
			devId := dev.IsolatedDeviceId
			devObj, err := IsolatedDeviceManager.FetchByIdOrName(userCred, devId)
			if err != nil {
				return errors.Wrapf(err, "fetch isolated device by %q", devId)
			}
			devType := devObj.(*SIsolatedDevice).DevType
			if !sets.NewString(api.VALID_CONTAINER_DEVICE_TYPES...).Has(devType) {
				return httperrors.NewInputParameterError("device type %s is not supported by container", devType)
			}
			dev.IsolatedDeviceId = devObj.GetId()
		} else if dev.Host != nil {
			if err := m.ValidateSpecHostDevice(dev.Host); err != nil {
				return errors.Wrap(err, "validate host device")
			}
		}
	}
	if spec.ImagePullPolicy == "" {
		spec.ImagePullPolicy = apis.ImagePullPolicyIfNotPresent
	}
	if !sets.NewString(apis.ImagePullPolicyAlways, apis.ImagePullPolicyIfNotPresent).Has(string(spec.ImagePullPolicy)) {
		return httperrors.NewInputParameterError("invalid image_pull_policy %s", spec.ImagePullPolicy)
	}

	if err := m.ValidateSpecVolumeMounts(ctx, userCred, pod, volumes, spec); err != nil {
		return errors.Wrap(err, "ValidateSpecVolumeMounts")
	}

	return nil
}

func (m *SContainerManager) ValidateSpecVolumeMounts(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, volumes []*api.PodVolume, spec *api.ContainerSpec) error {
	relation, err := m.GetVolumeMountRelations(ctx, userCred, pod, volumes, spec)
	if err != nil {
		return errors.Wrap(err, "GetVolumeMountRelations")
	}
	if _, _, err := m.ConvertVolumeMountRelationToSpec(relation); err != nil {
		return errors.Wrap(err, "ConvertVolumeMountRelationToSpec")
	}
	return nil
}

/*func (m *SContainerManager) GetContainerIndex(guestId string) (int, error) {
	cnt, err := m.Query("guest_id").Equals("guest_id", guestId).CountWithError()
	if err != nil {
		return -1, errors.Wrapf(err, "get container numbers of pod %s", guestId)
	}
	return cnt, nil
}

func (c *SContainer) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.ContainerCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal to ContainerCreateInput")
	}
	if input.Spec.ImagePullPolicy == "" {
		c.Spec.ImagePullPolicy = apis.ImagePullPolicyIfNotPresent
	}
	return nil
}*/

func (c *SContainer) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if !jsonutils.QueryBoolean(data, "skip_task", false) {
		if err := c.StartCreateTask(ctx, userCred, ""); err != nil {
			log.Errorf("StartCreateTask error: %v", err)
		}
	}
}

func (c *SContainer) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerCreateTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) GetPod() *SGuest {
	return GuestManager.FetchGuestById(c.GuestId)
}

func (m *SContainerManager) GetPodVolumes(ctx context.Context, userCred mcclient.TokenCredential, podId string) ([]*api.PodVolume, error) {
	pod := GuestManager.FetchGuestById(podId)
	if pod == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "Not found pod by id %s", podId)
	}
	return m.GetPodVolumesByObject(ctx, userCred, pod)
}

func (m *SContainerManager) GetPodVolumesByObject(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest) ([]*api.PodVolume, error) {
	cParams, err := pod.GetCreateParams(ctx, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "get pod create params")
	}
	return cParams.Pod.Volumes, nil
}

func (c *SContainer) GetPodVolumes(ctx context.Context, userCred mcclient.TokenCredential) ([]*api.PodVolume, error) {
	return GetContainerManager().GetPodVolumesByObject(ctx, userCred, c.GetPod())
}

func (c *SContainer) GetVolumeMounts() []*api.ContainerVolumeMount {
	return c.Spec.VolumeMounts
}

func (man *SContainerManager) GetVolumeByName(volumes []*api.PodVolume, name string) *api.PodVolume {
	for _, v := range volumes {
		if v.Name == name {
			return v
		}
	}
	return nil
}

type ContainerVolumeMountRelation struct {
	Name        string
	VolumeMount *api.ContainerVolumeMount
	Volume      *api.PodVolume

	pod *SGuest
}

func (vm *ContainerVolumeMountRelation) ToMountOrDevice() (*hostapi.ContainerDevice, *hostapi.ContainerMount, error) {
	if vm.Volume.Disk != nil {
		dev, err := vm.ToDiskDevice(vm.Volume.Disk)
		if err != nil {
			return nil, nil, errors.Wrap(err, "ToDiskDevice")
		}
		return dev, nil, nil
	}
	return nil, nil, nil
}

func (vm *ContainerVolumeMountRelation) ToDiskDevice(volDisk *api.PodVolumeDisk) (*hostapi.ContainerDevice, error) {
	// TODO: decouple this validation logic
	if vm.pod == nil {
		return nil, nil
	}
	disks, err := vm.pod.GetDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "Get pod %s disks", vm.pod.GetId())
	}
	if volDisk.DiskIndex >= len(disks) {
		return nil, errors.Errorf("disk_index %d is large than disk size %d", volDisk.DiskIndex, len(disks))
	}
	disk := disks[volDisk.DiskIndex]

	mount := vm.VolumeMount
	if !mount.AsRawDevice {
		return nil, errors.Errorf("Disk must be used as raw device")
	}
	var devType string
	switch disk.DiskFormat {
	case "raw":
		devType = api.CONTAINER_STORAGE_LOCAL_RAW
	default:
		return nil, errors.Errorf("unsupported disk format %s", disk.DiskFormat)
	}

	return &hostapi.ContainerDevice{
		Type:          devType,
		DiskId:        disk.GetId(),
		Path:          disk.AccessPath,
		ContainerPath: mount.RawDevicePath,
	}, nil
}

func (m *SContainerManager) GetVolumeMountRelations(ctx context.Context, userCred mcclient.TokenCredential, pod *SGuest, volumes []*api.PodVolume, spec *api.ContainerSpec) ([]*ContainerVolumeMountRelation, error) {
	relation := make([]*ContainerVolumeMountRelation, len(spec.VolumeMounts))
	for idx, vm := range spec.VolumeMounts {
		vol := GetContainerManager().GetVolumeByName(volumes, vm.Name)
		if vol == nil {
			return nil, errors.Wrapf(errors.ErrNotFound, "find pod volume by name %s", vm.Name)
		}
		relation[idx] = &ContainerVolumeMountRelation{
			Name:        vm.Name,
			VolumeMount: vm,
			Volume:      vol,
			pod:         pod,
		}
	}
	return relation, nil
}

func (c *SContainer) GetVolumeMountRelations(ctx context.Context, userCred mcclient.TokenCredential) ([]*ContainerVolumeMountRelation, error) {
	vols, err := c.GetPodVolumes(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetPodVolumes")
	}
	return GetContainerManager().GetVolumeMountRelations(ctx, userCred, c.GetPod(), vols, c.Spec)
}

func (c *SContainer) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_EXITED, api.CONTAINER_STATUS_START_FAILED).Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't start container in status %s", c.Status)
	}
	return nil, c.StartStartTask(ctx, userCred, "")
}

func (c *SContainer) StartStartTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_STARTING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStartTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.ContainerStopInput) (jsonutils.JSONObject, error) {
	if !sets.NewString(api.CONTAINER_STATUS_RUNNING, api.CONTAINER_STATUS_STOP_FAILED).Has(c.Status) {
		return nil, httperrors.NewInvalidStatusError("Can't stop container in status %s", c.Status)
	}
	return nil, c.StartStopTask(ctx, userCred, data, "")
}

func (c *SContainer) StartStopTask(ctx context.Context, userCred mcclient.TokenCredential, data *api.ContainerStopInput, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_STOPPING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerStopTask", c, userCred, jsonutils.Marshal(data).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_SYNC_STATUS, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerSyncStatusTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) error {
	return c.StartDeleteTask(ctx, userCred, "")
}

func (c *SContainer) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerDeleteTask", c, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) StartPullImageTask(ctx context.Context, userCred mcclient.TokenCredential, input *hostapi.ContainerPullImageInput, parentTaskId string) error {
	c.SetStatus(userCred, api.CONTAINER_STATUS_PULLING_IMAGE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ContainerPullImageTask", c, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (c *SContainer) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (m *SContainerManager) ConvertVolumeMountRelationToSpec(relation []*ContainerVolumeMountRelation) ([]*hostapi.ContainerDevice, []*hostapi.ContainerMount, error) {
	devs := make([]*hostapi.ContainerDevice, 0)
	mounts := make([]*hostapi.ContainerMount, 0)
	for _, r := range relation {
		dev, mount, err := r.ToMountOrDevice()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ToMountOrDevice: %#v", r)
		}
		if dev != nil {
			devs = append(devs, dev)
		}
		if mount != nil {
			mounts = append(mounts, mount)
		}
	}
	return devs, mounts, nil
}

func (c *SContainer) ToHostContainerSpec(ctx context.Context, userCred mcclient.TokenCredential) (*hostapi.ContainerSpec, error) {
	vmRelation, err := c.GetVolumeMountRelations(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetVolumeMountRelations")
	}
	ctrDevs, mounts, err := GetContainerManager().ConvertVolumeMountRelationToSpec(vmRelation)
	if err != nil {
		return nil, errors.Wrap(err, "ConvertVolumeRelationToSpec")
	}

	hSpec := &hostapi.ContainerSpec{
		ContainerSpec: c.Spec.ContainerSpec,
		Mounts:        mounts,
	}
	for _, dev := range c.Spec.Devices {
		if dev.IsolatedDeviceId != "" {
			isoDevObj, err := IsolatedDeviceManager.FetchById(dev.IsolatedDeviceId)
			if err != nil {
				return nil, errors.Wrapf(err, "Fetch isolated device by id", dev.IsolatedDeviceId)
			}
			isoDev := isoDevObj.(*SIsolatedDevice)
			ctrDev := &hostapi.ContainerDevice{
				IsolatedDeviceId: isoDev.GetId(),
				Type:             isoDev.DevType,
				Addr:             isoDev.Addr,
				Path:             isoDev.DevicePath,
			}
			ctrDevs = append(ctrDevs, ctrDev)
		} else if dev.Host != nil {
			hostDev := &hostapi.ContainerDevice{
				Type:          api.CONTAINER_DEV_HOST,
				Path:          dev.Host.HostPath,
				ContainerPath: dev.Host.ContainerPath,
				Permissions:   dev.Host.Permissions,
			}
			ctrDevs = append(ctrDevs, hostDev)
		}
	}
	hSpec.Devices = ctrDevs
	return hSpec, nil
}
