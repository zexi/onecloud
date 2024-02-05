package guestman

import (
	"context"

	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/pod"
)

type sPodGuestInstance struct {
	*sBaseGuestInstance
	cri pod.CRI
}

func newPodGuestInstance(id string, man *SGuestManager) *sPodGuestInstance {
	return &sPodGuestInstance{
		sBaseGuestInstance: newBaseGuestInstance(id, man, computeapi.HYPERVISOR_POD),
		cri:                man.GetCRI(),
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

func (s sPodGuestInstance) IsRunning() bool {
	s.cri
}
