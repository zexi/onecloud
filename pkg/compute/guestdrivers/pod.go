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

package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SPodDriver struct {
	SKVMGuestDriver
}

func init() {
	driver := SPodDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SPodDriver) newUnsupportOperationError(option string) error {
	return httperrors.NewUnsupportOperationError("Container not support %s", option)
}

func (self *SPodDriver) GetHypervisor() string {
	return api.HYPERVISOR_POD
}

func (self *SPodDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SPodDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if input.Pod == nil {
		return nil, httperrors.NewNotEmptyError("pod data is empty")
	}
	if len(input.Pod.Containers) == 0 {
		return nil, httperrors.NewNotEmptyError("containers data is empty")
	}
	for idx, ctr := range input.Pod.Containers {
		if err := self.validateContainerData(ctr); err != nil {
			return nil, errors.Wrapf(err, "data of %d container", idx)
		}
	}
	return input, nil
}

func (self *SPodDriver) validateContainerData(ctr *api.PodContainerCreateInput) error {
	return nil
}

func (self *SPodDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
	}
}

// for backward compatibility, deprecated driver
func (self *SPodDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_ON_PREMISE
	keys.Provider = api.CLOUD_PROVIDER_ONECLOUD
	keys.Brand = api.ONECLOUD_BRAND_ONECLOUD
	keys.Hypervisor = api.HYPERVISOR_POD
	return keys
}

func (self *SPodDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SPodDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SPodDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SPodDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SPodDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	header := self.getTaskRequestHeader(task)

	config := jsonutils.NewDict()
	desc, err := guest.GetDriver().GetJsonDescAtHost(ctx, userCred, guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, body, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	if jsonutils.QueryBoolean(body, "is_running", false) {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return body, nil
		})
	}
	return nil
}

func (self *SPodDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask, syncStatus bool) error {
	return self.newUnsupportOperationError("stop")
}

func (self *SPodDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return self.newUnsupportOperationError("suspend")
}

func (self *SPodDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return self.newUnsupportOperationError("soft reset")
}

func (self *SPodDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	return self.newUnsupportOperationError("detach disk")
}

func (self *SPodDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SPodDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, self.newUnsupportOperationError("VNC")
}

func (self *SPodDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	//guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	// do nothing here
	return nil
}

func (self *SPodDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SPodDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SPodDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, nil, false)
	return err
}

func (self *SPodDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	return jsonutils.Marshal(desc), nil
}

func (self *SPodDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	action, err := config.GetString("action")
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/servers/%s/%s", host.ManagerUri, guest.Id, action)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	return err
}

func (self *SPodDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// clean disk records in DB
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (self *SPodDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SPodDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return self.newUnsupportOperationError("create disk")
}

func (self *SPodDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, cpuSockets, vmemSize int64) error {
	return self.newUnsupportOperationError("change config")
}

func (self *SPodDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	return self.newUnsupportOperationError("rebuild root")
}

func (self *SPodDriver) GetRandomNetworkTypes() []string {
	return []string{api.NETWORK_TYPE_CONTAINER, api.NETWORK_TYPE_GUEST}
}

func (self *SPodDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SPodDriver) IsSupportGuestClone() bool {
	return false
}

func (self *SPodDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}

func (self *SPodDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return false, nil
}
