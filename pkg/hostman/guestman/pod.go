package guestman

import (
	"context"
	"fmt"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
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

func (s *sPodGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	log.Warningf("=================TBD")
	return nil, DeleteHomeDir(s)
}

func (s *sPodGuestInstance) ImportServer(pendingDelete bool) {
	log.Infof("======pod %s ImportServer do nothing", s.Id)
	// TODO: 参考SKVMGuestInstance，可以做更多的事，比如同步状态
	s.manager.SaveServer(s.Id, s)
	s.manager.RemoveCandidateServer(s)
}

func (s *sPodGuestInstance) DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (s *sPodGuestInstance) IsStopped() bool {
	//TODO implement me
	panic("implement me")
}

func (s *sPodGuestInstance) IsSuspend() bool {
	return false
}

func (s *sPodGuestInstance) getCRI() pod.CRI {
	return s.manager.GetCRI()
}

func (s *sPodGuestInstance) getPod(ctx context.Context) (*runtimeapi.PodSandbox, error) {
	pods, err := s.getCRI().ListPods(ctx, pod.ListPodOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "ListPods")
	}
	for _, p := range pods {
		if p.Metadata.Uid == s.Id {
			return p, nil
		}
	}
	return nil, errors.Errorf("Not found pod from containerd")
}

func (s *sPodGuestInstance) IsRunning() bool {
	_, err := s.getPod(context.Background())
	if err != nil {
		log.Warningf("check if pod of guest %s is running", s.GetName())
		return false
	}
	return true
	/*ctrs, err := s.getCRI().ListContainers(context.Background(), pod.ListContainerOptions{
		PodId: s.Id,
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
	return isAllRunning*/
}

func (s *sPodGuestInstance) HandleGuestStatus(ctx context.Context, status string, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	body.Set("status", jsonutils.NewString(status))
	hostutils.TaskComplete(ctx, body)
	return nil, nil
}

func (s *sPodGuestInstance) HandleGuestStart(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      s.GetDesc().Name,
			Uid:       s.GetId(),
			Namespace: s.GetDesc().TenantId,
			Attempt:   0,
		},
		Hostname:     s.GetDesc().Hostname,
		LogDirectory: "",
		DnsConfig:    nil,
		PortMappings: nil,
		Labels:       nil,
		Annotations:  nil,
		Linux:        nil,
		Windows:      nil,
	}
	ctrs := []*runtimeapi.ContainerConfig{
		{
			Metadata: &runtimeapi.ContainerMetadata{
				Name: fmt.Sprintf("%s-0", s.GetDesc().Name),
			},
			Image: &runtimeapi.ImageSpec{
				Image: "registry.cn-beijing.aliyuncs.com/yunionio/logger:v3.10.4",
			},
			Command: []string{
				"sleep",
				"360000",
			},
		},
	}
	resp, err := s.getCRI().RunContainers(ctx, podCfg, ctrs, "")
	if err != nil {
		return nil, errors.Wrap(err, "RunContainers")
	}
	log.Infof("======start resp: %s", jsonutils.Marshal(resp).PrettyString())
	return jsonutils.Marshal(resp), nil
}

func (s *sPodGuestInstance) LoadDesc() error {
	return LoadDesc(s)
}

func (s *sPodGuestInstance) PostLoad(m *SGuestManager) error {
	return nil
}
