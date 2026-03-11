package llm

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	llmutils "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMRestartTask struct {
	taskman.STask
}

type LLMResetTask struct {
	LLMRestartTask
}

func init() {
	taskman.RegisterTask(LLMRestartTask{})
	taskman.RegisterTask(LLMResetTask{})
}

func (task *LLMRestartTask) taskFailed(ctx context.Context, llm *models.SLLM, status string, err string) {
	if status == "" {
		status = api.LLM_STATUS_START_FAIL
	}
	llm.SetStatus(ctx, task.UserCred, status, err)
	db.OpsLog.LogEvent(llm, "restart", err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_VM_RESTART, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMRestartTask) taskComplete(ctx context.Context, llm *models.SLLM) {
	llm.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_RUNNING, "restart complete")
	task.SetStageComplete(ctx, nil)
}

func (task *LLMRestartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	llm.SetStatus(ctx, task.UserCred, computeapi.VM_START_STOP, "restarting")

	if task.HasParentTask() {
		task.OnSyncLLMInitStatusComplete(ctx, llm, nil)
		return
	}

	// Always do server syncstatus first (same pattern as CloudDesktopRestartTask)
	task.SetStage("OnSyncLLMInitStatusComplete", nil)
	if err := llm.StartSyncStatusTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "StartSyncStatusTask").Error())
		return
	}
}

func (task *LLMRestartTask) OnSyncLLMInitStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, err.String())
}

func (task *LLMRestartTask) OnSyncLLMInitStatusComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	srv, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetServer").Error())
		return
	}

	switch srv.Status {
	case computeapi.VM_RUNNING:
		task.SetStage("OnServerStopComplete", nil)
		if err := llm.StartLLMStopTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
			task.taskFailed(ctx, llm, api.LLM_STATUS_STOP_FAILED, errors.Wrap(err, "StartLLMStopTask").Error())
			return
		}
	case computeapi.VM_READY:
		task.OnServerStopComplete(ctx, llm, nil)
	default:
		if strings.Contains(srv.Status, "fail") {
			task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(errors.ErrInvalidStatus, srv.Status).Error())
			return
		}
		task.SetStage("OnSyncLLMInitStatusComplete", nil)
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := llm.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY, computeapi.VM_RUNNING}, 1800)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
			return nil, nil
		})
	}
}

func (task *LLMRestartTask) OnServerStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, api.LLM_STATUS_STOP_FAILED, err.String())
}

func (task *LLMRestartTask) OnServerStopComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	server, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetServer").Error())
		return
	}
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetLLMSku").Error())
		return
	}

	if sku.Cpu != server.VcpuCount || sku.Memory+1 != server.VmemSize {
		s := auth.GetSession(ctx, task.UserCred, "")
		params := computeapi.ServerChangeConfigInput{}
		params.VcpuCount = &sku.Cpu
		params.VmemSize = fmt.Sprintf("%dM", sku.Memory+1)
		_, err := compute.Servers.PerformAction(s, llm.CmpId, "change-config", jsonutils.Marshal(params))
		if err != nil {
			task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "change config").Error())
			return
		}

		task.SetStage("OnChangeConfigComplete", nil)
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := llm.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY}, 1800)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
			return nil, nil
		})
		return
	}

	task.OnChangeConfigComplete(ctx, llm, nil)
}

func (task *LLMRestartTask) OnChangeConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, err.String())
}

func (task *LLMRestartTask) OnChangeConfigComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetLLMSku").Error())
		return
	}
	volume, err := llm.GetVolume()
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetVolume").Error())
		return
	}

	targetSizeMB := 0
	if sku.Volumes != nil && !sku.Volumes.IsZero() {
		targetSizeMB = (*sku.Volumes)[0].SizeMB
	}

	if volume != nil && targetSizeMB > 0 && targetSizeMB > volume.SizeMB {
		task.SetStage("OnDiskResizeComplete", nil)
		_, err := volume.StartResizeTask(ctx, task.UserCred, api.VolumeResizeTaskInput{
			SizeMB:        targetSizeMB,
			DesktopStatus: api.STATUS_READY,
		}, task.GetTaskId())
		if err != nil {
			task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "StartResizeTask").Error())
			return
		}
		return
	}

	task.OnDiskResizeComplete(ctx, llm, nil)
}

func (task *LLMRestartTask) OnDiskResizeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, err.String())
}

func (task *LLMRestartTask) OnDiskResizeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	server, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetServer").Error())
		return
	}
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetLLMSku").Error())
		return
	}
	img, err := llm.GetLLMImage()
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "GetLLMImage").Error())
		return
	}
	volume, _ := llm.GetVolume()
	diskId := ""
	if volume != nil {
		diskId = volume.CmpId
	}

	drv := llm.GetLLMContainerDriver()
	specs := models.GetDriverPodContainers(ctx, drv, llm, img, sku, nil, server.IsolatedDevices, diskId)

	ctrNameToId, err := getGuestContainerNameToId(ctx, llm.CmpId)
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "getGuestContainers").Error())
		return
	}

	for _, spec := range specs {
		if spec == nil {
			continue
		}
		ctrId, ok := ctrNameToId[spec.Name]
		if !ok {
			task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrapf(errors.ErrNotFound, "container %s not found", spec.Name).Error())
			return
		}
		_, err := llmutils.UpdateContainer(ctx, ctrId, func(_ *computeapi.SContainer) *computeapi.ContainerSpec {
			return &spec.ContainerSpec
		})
		if err != nil {
			task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrapf(err, "UpdateContainer %s", spec.Name).Error())
			return
		}
	}

	task.SetStage("OnStartComplete", nil)
	s := auth.GetSession(ctx, task.GetUserCred(), "")
	err = s.WithTaskCallback(task.GetId(), func() error {
		_, err := compute.Servers.PerformAction(s, llm.CmpId, "start", nil)
		return err
	})
	if err != nil {
		task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, errors.Wrap(err, "server start").Error())
		return
	}
}

func (task *LLMRestartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskComplete(ctx, llm)
}

func (task *LLMRestartTask) OnStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, api.LLM_STATUS_START_FAIL, err.String())
}

func getGuestContainerNameToId(ctx context.Context, guestId string) (map[string]string, error) {
	s := auth.GetAdminSession(ctx, "")
	resp, err := compute.Containers.List(s, jsonutils.Marshal(map[string]string{
		"guest_id": guestId,
		"scope":    "max",
	}))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(resp.Data))
	for i := range resp.Data {
		id, _ := resp.Data[i].GetString("id")
		name, _ := resp.Data[i].GetString("name")
		if id == "" || name == "" {
			continue
		}
		out[name] = id
	}
	if len(out) == 0 {
		return nil, errors.Wrapf(errors.ErrNotFound, "no containers for guest %s", guestId)
	}
	return out, nil
}
