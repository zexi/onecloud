package guestman

import (
	"context"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pod"
)

type sPodGuestInstance struct {
	*sBaseGuestInstance
}

func newPodGuestInstance(id string, man *SGuestManager) *sPodGuestInstance {
	return &sPodGuestInstance{
		sBaseGuestInstance: newBaseGuestInstance(id, man, computeapi.HYPERVISOR_POD),
	}
}

func (s sPodGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s sPodGuestInstance) ImportServer(pendingDelete bool) {
	//TODO implement me
	panic("implement me")
}

func (s sPodGuestInstance) DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (s sPodGuestInstance) IsStopped() bool {
	//TODO implement me
	panic("implement me")
}

func (s sPodGuestInstance) IsSuspend() bool {
	//TODO implement me
	panic("implement me")
}

func (s sPodGuestInstance) getCRI() pod.CRI {
	return s.manager.GetCRI()
}

func (s sPodGuestInstance) IsRunning() bool {
	ctrs, err := s.getCRI().ListContainers(context.Background(), pod.ListContainerOptions{
		PodId: s.GetId(),
	})
	if err != nil {
		log.Errorf("List containers of pod %q", s.GetId())
		return false
	}
	// TODO: container s状态应该存在每个 container 资源里面
	// Pod 状态只放 guest 表
	isAllRunning := true
	for _, ctr := range ctrs {
		if ctr.State != runtimeapi.ContainerState_CONTAINER_RUNNING {
			isAllRunning = false
			break
		}
	}
	return isAllRunning
}
