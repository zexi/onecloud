package guestman

import (
	"context"
	"io/ioutil"
	"path"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/pod"
)

type PodInstance interface {
	GuestRuntimeInstance

	CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *computeapi.ContainerCreateInput) (jsonutils.JSONObject, error)
	StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error)
	DeleteContainer(ctx context.Context, cred mcclient.TokenCredential, id string) (jsonutils.JSONObject, error)
	SyncContainerStatus(ctx context.Context, cred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error)
}

type sContainer struct {
	Id    string `json:"id"`
	Index int    `json:"index"`
	CRIId string `json:"cri_id"`
}

func newContainer(id string, index int) *sContainer {
	return &sContainer{
		Id:    id,
		Index: index,
	}
}

type sPodGuestInstance struct {
	*sBaseGuestInstance
	containers map[string]*sContainer
}

func newPodGuestInstance(id string, man *SGuestManager) PodInstance {
	return &sPodGuestInstance{
		sBaseGuestInstance: newBaseGuestInstance(id, man, computeapi.HYPERVISOR_POD),
		containers:         make(map[string]*sContainer),
	}
}

func (s *sPodGuestInstance) CleanGuest(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	criId := s.getCRIId()
	if err := s.getCRI().RemovePod(ctx, criId); err != nil {
		return nil, errors.Wrapf(err, "RemovePod with cri_id %q", criId)
	}
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
		log.Warningf("check if pod of guest %s is running", s.Id)
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
	go func() {
		resp, err := s.startPod(context.Background(), userCred)
		if err != nil {
			hostutils.TaskFailed(ctx, err.Error())
			return
		}
		hostutils.TaskComplete(ctx, jsonutils.Marshal(resp))
	}()
	return nil, nil
}

func (s *sPodGuestInstance) startPod(ctx context.Context, userCred mcclient.TokenCredential) (*computeapi.PodStartResponse, error) {
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      s.GetDesc().Name,
			Uid:       s.GetId(),
			Namespace: s.GetDesc().TenantId,
			Attempt:   1,
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
	criId, err := s.getCRI().RunPod(ctx, podCfg, "")
	if err != nil {
		return nil, errors.Wrap(err, "cri.RunPod")
	}
	if err := s.setCRIInfo(ctx, userCred, criId, podCfg); err != nil {
		return nil, errors.Wrap(err, "setCRIId")
	}
	return &computeapi.PodStartResponse{
		CRIId:     criId,
		IsRunning: false,
	}, nil
}

func (s *sPodGuestInstance) LoadDesc() error {
	if err := LoadDesc(s); err != nil {
		return errors.Wrap(err, "LoadDesc")
	}
	if err := s.loadContainers(); err != nil {
		return errors.Wrap(err, "loadContainers")
	}
	return nil
}

func (s *sPodGuestInstance) loadContainers() error {
	ctrFile := s.getContainersFilePath()
	ctrStr, err := ioutil.ReadFile(ctrFile)
	if err != nil {
		return errors.Wrapf(err, "read %s", ctrFile)
	}
	obj, err := jsonutils.Parse(ctrStr)
	if err != nil {
		return errors.Wrapf(err, "jsonutils.Parse %s", ctrStr)
	}
	ctrs := make(map[string]*sContainer)
	if err := obj.Unmarshal(ctrs); err != nil {
		return errors.Wrapf(err, "unmarshal %s to container map", obj.String())
	}
	log.Infof("========load containers: %#v", ctrs)
	s.containers = ctrs
	return nil
}

func (s *sPodGuestInstance) PostLoad(m *SGuestManager) error {
	return nil
}

func (s *sPodGuestInstance) getContainerCRIId(ctrId string) (string, error) {
	ctr := s.getContainer(ctrId)
	if ctr == nil {
		return "", errors.Errorf("Not found container %s", ctrId)
	}
	return ctr.CRIId, nil
}

func (s *sPodGuestInstance) StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	go func() {
		criId, err := s.getContainerCRIId(ctrId)
		if err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "get container cri id").Error())
			return
		}
		if err := s.getCRI().StartContainer(context.Background(), criId); err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "CRI.StartContainer").Error())
			return
		}
		hostutils.TaskComplete(ctx, nil)
	}()
	return nil, nil
}

func (s *sPodGuestInstance) getCRIId() string {
	return s.GetSourceDesc().Metadata[computeapi.POD_METADATA_CRI_ID]
}

func (s *sPodGuestInstance) setCRIInfo(ctx context.Context, userCred mcclient.TokenCredential, criId string, cfg *runtimeapi.PodSandboxConfig) error {
	s.Desc.Metadata[computeapi.POD_METADATA_CRI_ID] = criId
	cfgStr := jsonutils.Marshal(cfg).String()
	s.Desc.Metadata[computeapi.POD_METADATA_CRI_CONFIG] = cfgStr

	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Servers.SetMetadata(session, s.GetId(), jsonutils.Marshal(map[string]string{
		computeapi.POD_METADATA_CRI_ID:     criId,
		computeapi.POD_METADATA_CRI_CONFIG: cfgStr,
	})); err != nil {
		return errors.Wrapf(err, "set cri_id of pod %s", s.GetId())
	}
	return SaveDesc(s, s.Desc)
}

func (s *sPodGuestInstance) setContainerCRIInfo(ctx context.Context, userCred mcclient.TokenCredential, ctrId, criId string) error {
	session := auth.GetSession(ctx, userCred, options.HostOptions.Region)
	if _, err := computemod.Containers.SetMetadata(session, ctrId, jsonutils.Marshal(map[string]string{
		computeapi.CONTAINER_METADATA_CRI_ID: criId,
	})); err != nil {
		return errors.Wrapf(err, "set cri_id of container %s", ctrId)
	}
	// TODO: 添加本地容器和 cri id 对应关系
	return nil
}

func (s *sPodGuestInstance) getPodSandboxConfig() (*runtimeapi.PodSandboxConfig, error) {
	cfgStr := s.Desc.Metadata[computeapi.POD_METADATA_CRI_CONFIG]
	obj, err := jsonutils.ParseString(cfgStr)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseString to json object: %s", cfgStr)
	}
	podCfg := new(runtimeapi.PodSandboxConfig)
	if err := obj.Unmarshal(podCfg); err != nil {
		return nil, errors.Wrap(err, "Unmarshal to PodSandboxConfig")
	}
	return podCfg, nil
}

func (s *sPodGuestInstance) saveContainer(id string, index int, criId string) error {
	_, ok := s.containers[id]
	if ok {
		return errors.Errorf("container %s already exists", criId)
	}
	ctr := newContainer(id, index)
	ctr.CRIId = criId
	s.containers[id] = ctr
	if err := s.saveContainersFile(s.containers); err != nil {
		return errors.Wrap(err, "saveContainersFile")
	}
	return nil
}

func (s *sPodGuestInstance) saveContainersFile(containers map[string]*sContainer) error {
	content := jsonutils.Marshal(containers).String()
	if err := fileutils2.FilePutContents(s.getContainersFilePath(), content, false); err != nil {
		return errors.Wrapf(err, "put content %s to containers file", content)
	}
	return nil
}

func (s *sPodGuestInstance) getContainersFilePath() string {
	return path.Join(s.HomeDir(), "containers")
}

func (s *sPodGuestInstance) getContainer(id string) *sContainer {
	return s.containers[id]
}

func (s *sPodGuestInstance) CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *computeapi.ContainerCreateInput) (jsonutils.JSONObject, error) {
	go func() {
		ctrCriId, err := s.createContainer(context.Background(), userCred, id, input)
		if err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "CRI.CreateContainer").Error())
			return
		}
		if err := s.setContainerCRIInfo(ctx, userCred, id, ctrCriId); err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "setContainerCRIInfo").Error())
			return
		}
		hostutils.TaskComplete(ctx, nil)
	}()
	return nil, nil
}

func (s *sPodGuestInstance) createContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.ContainerCreateInput) (string, error) {
	spec := input.Spec
	ctrCfg := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: input.Name,
		},
		Image: &runtimeapi.ImageSpec{
			Image: spec.Image,
		},
		Linux: &runtimeapi.LinuxContainerConfig{},
	}
	if len(spec.Command) != 0 {
		ctrCfg.Command = spec.Command
	}
	if len(spec.Args) != 0 {
		ctrCfg.Args = spec.Args
	}
	podCfg, err := s.getPodSandboxConfig()
	if err != nil {
		return "", errors.Wrap(err, "getPodSandboxConfig")
	}
	criId, err := s.getCRI().CreateContainer(ctx, s.getCRIId(), podCfg, ctrCfg, true)
	if err != nil {
		return "", errors.Wrap(err, "cri.CreateContainer")
	}
	// TODO: 设置 index
	if err := s.saveContainer(ctrId, 0, criId); err != nil {
		return "", errors.Wrap(err, "saveContainer")
	}
	return criId, nil
}

func (s *sPodGuestInstance) DeleteContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "getContainerCRIId")
	}
	go func() {
		if err := s.deleteContainer(context.Background(), ctrId, criId); err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "deleteContainer").Error())
			return
		}
		hostutils.TaskComplete(ctx, nil)
	}()
	return nil, nil
}

func (s *sPodGuestInstance) deleteContainer(ctx context.Context, ctrId string, criId string) error {
	if err := s.getCRI().RemoveContainer(ctx, criId); err != nil {
		return errors.Wrap(err, "cri.RemoveContainer")
	}
	// refresh local containers file
	delete(s.containers, ctrId)
	if err := s.saveContainersFile(s.containers); err != nil {
		return errors.Wrap(err, "saveContainersFile")
	}
	return nil
}

func (s *sPodGuestInstance) SyncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	go func() {
		resp, err := s.syncContainerStatus(context.Background(), userCred, ctrId)
		if err != nil {
			hostutils.TaskFailed(ctx, errors.Wrap(err, "sync container status").Error())
			return
		}
		hostutils.TaskComplete(ctx, resp)
	}()
	return nil, nil
}

func (s *sPodGuestInstance) syncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrapf(err, "get container cri_id by %s", ctrId)
	}
	resp, err := s.getCRI().ContainerStatus(ctx, criId)
	if err != nil {
		return nil, errors.Wrap(err, "cri.ContainerStatus")
	}
	status := computeapi.CONTAINER_STATUS_UNKNOWN
	switch resp.Status.State {
	case runtimeapi.ContainerState_CONTAINER_CREATED:
		status = computeapi.CONTAINER_STATUS_CREATED
	case runtimeapi.ContainerState_CONTAINER_RUNNING:
		status = computeapi.CONTAINER_STATUS_RUNNING
	case runtimeapi.ContainerState_CONTAINER_EXITED:
		status = computeapi.CONTAINER_STATUS_EXITED
	case runtimeapi.ContainerState_CONTAINER_UNKNOWN:
		status = computeapi.CONTAINER_STATUS_UNKNOWN
	}
	return jsonutils.Marshal(computeapi.ContainerSyncStatusResponse{Status: status}), nil
}
