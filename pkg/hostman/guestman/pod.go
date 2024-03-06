package guestman

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	losetup "github.com/zexi/golosetup"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/pod"
)

type PodInstance interface {
	GuestRuntimeInstance

	CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error)
	StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error)
	DeleteContainer(ctx context.Context, cred mcclient.TokenCredential, id string) (jsonutils.JSONObject, error)
	SyncContainerStatus(ctx context.Context, cred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error)
	StopContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PullImage(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error)
}

type sContainer struct {
	Id    string `json:"id"`
	Index int    `json:"index"`
	CRIId string `json:"cri_id"`
}

func newContainer(id string) *sContainer {
	return &sContainer{
		Id: id,
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
	if err := s.cleanVolumes(); err != nil {
		return nil, errors.Wrap(err, "clean volumes")
	}
	return nil, DeleteHomeDir(s)
}

func (s *sPodGuestInstance) cleanVolumes() error {
	disks := s.GetDesc().Disks
	input, err := s.getPodCreateParams()
	if err != nil {
		return errors.Wrapf(err, "getPodCreateParams")
	}
	vols := input.Volumes
	for _, vol := range vols {
		if err := s.cleanVolume(vol, disks); err != nil {
			return errors.Wrapf(err, "clean volume %#v", vol)
		}
	}
	return nil
}

func (s *sPodGuestInstance) cleanVolume(vol *computeapi.PodVolume, disks []*desc.SGuestDisk) error {
	// TODO: use interface or move this logic to disk deletion
	if vol.Disk != nil {
		if err := s.cleanVolumeDisk(vol.Disk, disks); err != nil {
			return errors.Wrapf(err, "clean disk volume %s", vol.Name)
		}
	}
	return nil
}

func (s *sPodGuestInstance) cleanVolumeDisk(vol *computeapi.PodVolumeDisk, disks []*desc.SGuestDisk) error {
	disk := disks[vol.DiskIndex]
	devs, err := losetup.ListDevices()
	if err != nil {
		return errors.Wrap(err, "ListDevices")
	}
	for _, dev := range devs.LoopDevs {
		if !strings.Contains(dev.BackFile, filepath.Base(disk.Path)) {
			continue
		}
		log.Infof("start detach loop device %#v", dev)
		if err := losetup.DetachDevice(dev.Name); err != nil {
			return errors.Wrapf(err, "detach loop device %s", dev.Name)
		}
	}
	return nil
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
	return nil, errors.Wrap(httperrors.ErrNotFound, "Not found pod from containerd")
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
	hostutils.DelayTask(ctx, func(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
		resp, err := s.startPod(context.Background(), userCred)
		if err != nil {
			return nil, errors.Wrap(err, "startPod")
		}
		return jsonutils.Marshal(resp), nil
	}, nil)
	return nil, nil
}

func (s *sPodGuestInstance) getCreateParams() (jsonutils.JSONObject, error) {
	createParamsStr, ok := s.GetDesc().Metadata[computeapi.VM_METADATA_CREATE_PARAMS]
	if !ok {
		return nil, errors.Errorf("not found %s in metadata", computeapi.VM_METADATA_CREATE_PARAMS)
	}
	return jsonutils.ParseString(createParamsStr)
}

func (s *sPodGuestInstance) getPodCreateParams() (*computeapi.PodCreateInput, error) {
	createParams, err := s.getCreateParams()
	if err != nil {
		return nil, errors.Wrapf(err, "getCreateParams")
	}
	input := new(computeapi.PodCreateInput)
	if err := createParams.Unmarshal(input, "pod"); err != nil {
		return nil, errors.Wrapf(err, "unmarshal to pod creation input")
	}
	return input, nil
}

func (s *sPodGuestInstance) getPodLogDir() string {
	return filepath.Join(s.HomeDir(), "logs")
}

func (s *sPodGuestInstance) startPod(ctx context.Context, userCred mcclient.TokenCredential) (*computeapi.PodStartResponse, error) {
	podInput, err := s.getPodCreateParams()
	if err != nil {
		return nil, errors.Wrap(err, "getPodCreateParams")
	}
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      s.GetDesc().Name,
			Uid:       s.GetId(),
			Namespace: s.GetDesc().TenantId,
			Attempt:   1,
		},
		Hostname:     s.GetDesc().Hostname,
		LogDirectory: s.getPodLogDir(),
		DnsConfig:    nil,
		PortMappings: nil,
		Labels:       nil,
		Annotations:  nil,
		Linux: &runtimeapi.LinuxPodSandboxConfig{
			CgroupParent: "",
			SecurityContext: &runtimeapi.LinuxSandboxSecurityContext{
				NamespaceOptions:   nil,
				SelinuxOptions:     nil,
				RunAsUser:          nil,
				RunAsGroup:         nil,
				ReadonlyRootfs:     false,
				SupplementalGroups: nil,
				//Privileged:         true,
				Seccomp:            nil,
				Apparmor:           nil,
				SeccompProfilePath: "",
			},
			Sysctls: nil,
		},
		Windows: nil,
	}

	if len(podInput.PortMappings) != 0 {
		podCfg.PortMappings = make([]*runtimeapi.PortMapping, len(podInput.PortMappings))
		for idx := range podInput.PortMappings {
			pm := podInput.PortMappings[idx]
			runtimePm := &runtimeapi.PortMapping{
				ContainerPort: pm.ContainerPort,
				HostPort:      pm.HostPort,
				HostIp:        pm.HostIp,
			}
			switch pm.Protocol {
			case computeapi.PodPortMappingProtocolTCP:
				runtimePm.Protocol = runtimeapi.Protocol_TCP
			case computeapi.PodPortMappingProtocolUDP:
				runtimePm.Protocol = runtimeapi.Protocol_UDP
			case computeapi.PodPortMappingProtocolSCTP:
				runtimePm.Protocol = runtimeapi.Protocol_SCTP
			default:
				return nil, errors.Errorf("invalid protocol: %q", pm.Protocol)
			}
			podCfg.PortMappings[idx] = runtimePm
		}
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
	s.containers = make(map[string]*sContainer)
	ctrFile := s.getContainersFilePath()
	if !fileutils2.Exists(ctrFile) {
		log.Warningf("pod %s containers file %s doesn't exist", s.Id, ctrFile)
		return nil
	}
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
	s.containers = ctrs
	return nil
}

func (s *sPodGuestInstance) PostLoad(m *SGuestManager) error {
	return nil
}

func (s *sPodGuestInstance) getContainerCRIId(ctrId string) (string, error) {
	ctr := s.getContainer(ctrId)
	if ctr == nil {
		return "", errors.Wrapf(errors.ErrNotFound, "Not found container %s", ctrId)
	}
	return ctr.CRIId, nil
}

func (s *sPodGuestInstance) StartContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error) {
	_, hasCtr := s.containers[ctrId]
	needRecreate := false
	if hasCtr {
		status, err := s.getContainerStatus(ctx, ctrId)
		if err != nil {
			return nil, errors.Wrap(err, "get container status")
		}
		if status == computeapi.CONTAINER_STATUS_EXITED {
			needRecreate = true
		} else if status != computeapi.CONTAINER_STATUS_CREATED {
			return nil, errors.Wrapf(err, "can't start container when status is %s", status)
		}
	}
	if !hasCtr || needRecreate {
		log.Infof("recreate container %s before starting. hasCtr: %v, needRecreate: %v", ctrId, hasCtr, needRecreate)
		// delete and recreate the container before starting
		if hasCtr {
			if _, err := s.DeleteContainer(ctx, userCred, ctrId); err != nil {
				return nil, errors.Wrap(err, "delete container before starting")
			}
		}
		if _, err := s.CreateContainer(ctx, userCred, ctrId, input); err != nil {
			return nil, errors.Wrap(err, "recreate container before starting")
		}
	}

	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	if err := s.getCRI().StartContainer(ctx, criId); err != nil {
		return nil, errors.Wrap(err, "CRI.StartContainer")
	}
	return nil, nil
}

func (s *sPodGuestInstance) StopContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container cri id")
	}
	var timeout int64 = 0
	if body.Contains("timeout") {
		timeout, _ = body.Int("timeout")
	}
	if err := s.getCRI().StopContainer(context.Background(), criId, timeout); err != nil {
		return nil, errors.Wrap(err, "CRI.StopContainer")
	}
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
	return nil
}

func (s *sPodGuestInstance) getPodSandboxConfig() (*runtimeapi.PodSandboxConfig, error) {
	cfgStr := s.GetSourceDesc().Metadata[computeapi.POD_METADATA_CRI_CONFIG]
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

func (s *sPodGuestInstance) saveContainer(id string, criId string) error {
	_, ok := s.containers[id]
	if ok {
		return errors.Errorf("container %s already exists", criId)
	}
	ctr := newContainer(id)
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

func (s *sPodGuestInstance) CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, id string, input *hostapi.ContainerCreateInput) (jsonutils.JSONObject, error) {
	ctrCriId, err := s.createContainer(ctx, userCred, id, input)
	if err != nil {
		return nil, errors.Wrap(err, "CRI.CreateContainer")
	}
	if err := s.setContainerCRIInfo(ctx, userCred, id, ctrCriId); err != nil {
		return nil, errors.Wrap(err, "setContainerCRIInfo")
	}
	return nil, nil
}

func (s *sPodGuestInstance) getContainerLogPath(ctrId string) string {
	return filepath.Join(fmt.Sprintf("%s.log", ctrId))
}

func (s *sPodGuestInstance) createContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerCreateInput) (string, error) {
	log.Infof("=====container input: %s", jsonutils.Marshal(input).PrettyString())
	spec := input.Spec
	ctrCfg := &runtimeapi.ContainerConfig{
		Metadata: &runtimeapi.ContainerMetadata{
			Name: input.Name,
		},
		Image: &runtimeapi.ImageSpec{
			Image: spec.Image,
		},
		Linux: &runtimeapi.LinuxContainerConfig{
			SecurityContext: &runtimeapi.LinuxContainerSecurityContext{
				Capabilities: &runtimeapi.Capability{
					AddCapabilities: []string{"SYS_ADMIN"},
				},
				//Privileged:         true,
				NamespaceOptions:   nil,
				SelinuxOptions:     nil,
				RunAsUser:          nil,
				RunAsGroup:         nil,
				RunAsUsername:      "",
				ReadonlyRootfs:     false,
				SupplementalGroups: nil,
				NoNewPrivs:         false,
				MaskedPaths:        nil,
				ReadonlyPaths:      nil,
				Seccomp:            nil,
				Apparmor:           nil,
				ApparmorProfile:    "",
				SeccompProfilePath: "",
			},
		},
		LogPath: s.getContainerLogPath(ctrId),
		Envs:    []*runtimeapi.KeyValue{},
		Devices: []*runtimeapi.Device{},
	}
	if len(spec.Devices) != 0 {
		for _, dev := range spec.Devices {
			if dev.Type == computeapi.CONTAINER_DEV_HOST {
				ctrCfg.Devices = append(ctrCfg.Devices, &runtimeapi.Device{
					ContainerPath: dev.ContainerPath,
					HostPath:      dev.Path,
					Permissions:   dev.Permissions,
				})
			} else {
				man, err := isolated_device.GetContainerDeviceManager(isolated_device.ContainerDeviceType(dev.Type))
				if err != nil {
					return "", errors.Wrapf(err, "GetContainerDeviceManager by type %q", dev.Type)
				}
				ctrDevs, err := man.NewContainerDevices(dev)
				if err != nil {
					return "", errors.Wrapf(err, "NewContainerDevices with %#v", dev)
				}
				ctrCfg.Devices = append(ctrCfg.Devices, ctrDevs...)
			}
		}
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
	criId, err := s.getCRI().CreateContainer(ctx, s.getCRIId(), podCfg, ctrCfg, false)
	if err != nil {
		return "", errors.Wrap(err, "cri.CreateContainer")
	}
	if err := s.saveContainer(ctrId, criId); err != nil {
		return "", errors.Wrap(err, "saveContainer")
	}
	return criId, nil
}

func (s *sPodGuestInstance) DeleteContainer(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		return nil, errors.Wrap(err, "getContainerCRIId")
	}
	if criId != "" {
		if err := s.getCRI().RemoveContainer(ctx, criId); err != nil {
			return nil, errors.Wrap(err, "cri.RemoveContainer")
		}
	}
	// refresh local containers file
	delete(s.containers, ctrId)
	if err := s.saveContainersFile(s.containers); err != nil {
		return nil, errors.Wrap(err, "saveContainersFile")
	}
	return nil, nil
}

func (s *sPodGuestInstance) getContainerStatus(ctx context.Context, ctrId string) (string, error) {
	criId, err := s.getContainerCRIId(ctrId)
	if err != nil {
		return "", errors.Wrapf(err, "get container cri_id by %s", ctrId)
	}
	resp, err := s.getCRI().ContainerStatus(ctx, criId)
	if err != nil {
		return "", errors.Wrap(err, "cri.ContainerStatus")
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
	return status, nil
}

func (s *sPodGuestInstance) SyncContainerStatus(ctx context.Context, userCred mcclient.TokenCredential, ctrId string) (jsonutils.JSONObject, error) {
	status, err := s.getContainerStatus(ctx, ctrId)
	if err != nil {
		return nil, errors.Wrap(err, "get container status")
	}
	return jsonutils.Marshal(computeapi.ContainerSyncStatusResponse{Status: status}), nil
}

func (s *sPodGuestInstance) PullImage(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *hostapi.ContainerPullImageInput) (jsonutils.JSONObject, error) {
	policy := input.PullPolicy
	if policy == apis.ImagePullPolicyIfNotPresent || policy == "" {
		// check if image is presented
		img, err := s.getCRI().ImageStatus(ctx, &runtimeapi.ImageStatusRequest{
			Image: &runtimeapi.ImageSpec{
				Image: input.Image,
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "cri.ImageStatus %s", input.Image)
		}
		if img.Image != nil {
			log.Infof("image %s already exists, skipping pulling it when policy is %s", input.Image, policy)
			return jsonutils.Marshal(&runtimeapi.PullImageResponse{
				ImageRef: img.Image.Id,
			}), nil
		}
	}
	podCfg, err := s.getPodSandboxConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get pod sandbox config")
	}
	req := &runtimeapi.PullImageRequest{
		Image: &runtimeapi.ImageSpec{
			Image: input.Image,
		},
		SandboxConfig: podCfg,
	}
	if input.Auth != nil {
		authCfg := &runtimeapi.AuthConfig{
			Username:      input.Auth.Username,
			Password:      input.Auth.Password,
			Auth:          input.Auth.Auth,
			ServerAddress: input.Auth.ServerAddress,
			IdentityToken: input.Auth.IdentityToken,
			RegistryToken: input.Auth.RegistryToken,
		}
		req.Auth = authCfg
	}
	resp, err := s.getCRI().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "cri.PullImage %s", input.Image)
	}
	return jsonutils.Marshal(resp), nil
}
