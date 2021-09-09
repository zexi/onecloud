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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(GuestDetachIsolatedDeviceTask{})
}

type GuestDetachIsolatedDeviceTask struct {
	SGuestBaseTask
}

func (t *GuestDetachIsolatedDeviceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	dev, err := t.getDevice()
	if err != nil {
		t.setStageFailed(ctx, err)
		return
	}

	data = jsonutils.Marshal(map[string]interface{}{
		"device_id": dev.GetId(),
	})

	t.SetStage("OnDetachComplete", nil)
	if err := guest.GetDriver().RequestDetachIsolatedDevice(ctx, guest, t.GetUserCred(), data.(*jsonutils.JSONDict), t); err != nil {
		t.setStageFailed(ctx, err)
		return
	}
}

func (t *GuestDetachIsolatedDeviceTask) setStageFailed(ctx context.Context, err error) {
	t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (t *GuestDetachIsolatedDeviceTask) getDevice() (*models.SIsolatedDevice, error) {
	devId, err := t.GetParams().GetString("device_id")
	if err != nil {
		return nil, err
	}
	obj, err := models.IsolatedDeviceManager.FetchById(devId)
	if err != nil {
		return nil, err
	}
	return obj.(*models.SIsolatedDevice), nil
}

func (t *GuestDetachIsolatedDeviceTask) OnDetachComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {

}

func (t *GuestDetachIsolatedDeviceTask) OnDetachCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {

}
