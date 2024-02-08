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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type PodCreateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(PodCreateTask{})
}

func (t *PodCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.SetStage("OnPodCreated", nil)
	t.OnWaitPodCreated(ctx, obj, nil)
}

func (t *PodCreateTask) OnWaitPodCreated(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateTask", obj, t.GetUserCred(), t.GetParams(), t.GetTaskId(), "", nil)
	if err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("New GuestCreateTask")))
		return
	}
	if err := task.ScheduleRun(nil); err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (t *PodCreateTask) OnPodCreated(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.SetStage("OnContainerCreated", nil)
	input, err := guest.GetCreateParams(ctx, t.GetUserCred())
	if err != nil {
		t.onError(ctx, errors.Wrap(err, "GetCreateParams"))
		return
	}

	ctrs := make([]*models.SContainer, len(input.Pod.Containers))
	for idx, ctr := range input.Pod.Containers {
		if obj, err := models.GetContainerManager().CreateOnPod(ctx, t.GetUserCred(), guest.GetOwnerId(), guest, ctr); err != nil {
			t.onError(ctx, errors.Wrapf(err, "create container on pod: %s", guest.GetName()))
			return
		} else {
			ctrs[idx] = obj
		}
	}

	for idx, ctr := range ctrs {
		if err := ctr.StartCreateTask(ctx, t.GetUserCred(), t.GetTaskId()); err != nil {
			t.onError(ctx, errors.Wrapf(err, "start container %d creation task", idx))
			return
		}
	}
}

func (t *PodCreateTask) onError(ctx context.Context, err error) {
	t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (t *PodCreateTask) OnPodCreatedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.SetStageFailed(ctx, data)
}

func (t *PodCreateTask) OnContainerCreated(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	ctrs, err := models.GetContainerManager().GetContainersByPod(guest.GetId())
	if err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	isAllRunning := true
	for _, ctr := range ctrs {
		if ctr.GetStatus() != api.CONTAINER_STATUS_RUNNING {
			isAllRunning = false
		}
	}
	if isAllRunning {
		t.SetStageComplete(ctx, nil)
	}
}

func (t *PodCreateTask) OnContainerCreatedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.SetStageFailed(ctx, data)
}
