package guestman

import (
	"context"

	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type sPodGuestInstance struct {
	*sBaseGuestInstance
}

func (s sPodGuestInstance) IsRunning() bool {
	//TODO implement me
	panic("implement me")
}

func newPodGuestInstance(id string, man *SGuestManager) *sPodGuestInstance {
	return &sPodGuestInstance{
		sBaseGuestInstance: newBaseGuestInstance(id, man, computeapi.HYPERVISOR_POD),
	}
}

func (s sPodGuestInstance) SaveDesc(guestDesc *desc.SGuestDesc) error {
	//TODO implement me
	panic("implement me")
}

func (s sPodGuestInstance) DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	//TODO implement me
	panic("implement me")
}
