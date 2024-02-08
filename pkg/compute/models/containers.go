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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
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
	// Index is the container order of a pod
	Index int `nullable:"false" default:"0" list:"user" update:"user"`
	// Spec stores all container running options
	Spec api.ContainerSpec `length:"long"`
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
	input.GuestId = obj.GetId()
	if err := m.validateSpec(input.Spec); err != nil {
		return nil, errors.Wrap(err, "validate spec")
	}
	return input, nil
}

func (m *SContainerManager) validateSpec(input api.ContainerSpec) error {
	return nil
}

func (m *SContainerManager) GetContainerIndex(guestId string) (int, error) {
	cnt, err := m.Query("guest_id").Equals("guest_id", guestId).CountWithError()
	if err != nil {
		return -1, errors.Wrapf(err, "get container numbers of pod %s", guestId)
	}
	return cnt, nil
}

func (c *SContainer) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// calculate index
	idx, err := GetContainerManager().GetContainerIndex(c.GuestId)
	if err != nil {
		return errors.Wrap(err, "GetContainerIndex")
	}
	c.Index = idx
	return nil
}

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
