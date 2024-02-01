package main

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type CRICtl interface {
	RunContainers(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, containerConfigs []*runtimeapi.ContainerConfig, runtimeHandler string) (*RunContainersResponse, error)
	ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error)
}

type crictl struct {
	endpoint string
	timeout  time.Duration
	conn     *grpc.ClientConn

	imgCli runtimeapi.ImageServiceClient
	runCli runtimeapi.RuntimeServiceClient
}

func NewCRICtl(endpoint string, timeout time.Duration) (CRICtl, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var dialOpts []grpc.DialOption
	maxMsgSize := 1024 * 1024 * 16
	dialOpts = append(dialOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)))

	conn, err := grpc.DialContext(ctx, endpoint, dialOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Connect remote endpoint %q failed", endpoint)
	}

	imgCli := runtimeapi.NewImageServiceClient(conn)
	runCli := runtimeapi.NewRuntimeServiceClient(conn)

	return &crictl{
		endpoint: endpoint,
		timeout:  timeout,
		conn:     conn,
		imgCli:   imgCli,
		runCli:   runCli,
	}, nil
}

func (c crictl) getImageClient() runtimeapi.ImageServiceClient {
	return c.imgCli
}

func (c crictl) getRuntimeClient() runtimeapi.RuntimeServiceClient {
	return c.runCli
}

func (c crictl) ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.getImageClient().ListImages(ctx, &runtimeapi.ListImagesRequest{
		Filter: filter,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "ListImages with filter %s", filter.String())
	}
	return resp.Images, nil
}

func (c crictl) RunPodSandbox(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error) {
	req := &runtimeapi.RunPodSandboxRequest{
		Config:         podConfig,
		RuntimeHandler: runtimeHandler,
	}
	log.Infof("RunPodSandboxRequest: %v", req)
	r, err := c.getRuntimeClient().RunPodSandbox(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "RunPodSandbox with request: %s", req.String())
	}
	return r.GetPodSandboxId(), nil
}

// PullImageWithSandbox sends a PullImageRequest to the server and parses
// the returned PullImageResponses.
func (c crictl) PullImageWithSandbox(ctx context.Context, image string, auth *runtimeapi.AuthConfig, sandbox *runtimeapi.PodSandboxConfig, ann map[string]string) (*runtimeapi.PullImageResponse, error) {
	req := &runtimeapi.PullImageRequest{
		Image: &runtimeapi.ImageSpec{
			Image:              image,
			Annotations:        ann,
			UserSpecifiedImage: "",
			RuntimeHandler:     "",
		},
		Auth:          auth,
		SandboxConfig: sandbox,
	}
	log.Infof("PullImageRequest: %v", req)
	r, err := c.getImageClient().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "PullImage with %s", req)
	}
	return r, nil
}

// CreateContainer sends a CreateContainerRequest to the server, and parses
// the returned CreateContainerResponse.
func (c crictl) CreateContainer(ctx context.Context,
	podId string, podConfig *runtimeapi.PodSandboxConfig,
	ctrConfig *runtimeapi.ContainerConfig, withPull bool) (string, error) {
	req := &runtimeapi.CreateContainerRequest{
		PodSandboxId:  podId,
		Config:        ctrConfig,
		SandboxConfig: podConfig,
	}

	image := ctrConfig.GetImage().GetImage()
	if ctrConfig.Image.UserSpecifiedImage == "" {
		ctrConfig.Image.UserSpecifiedImage = image
	}

	// When there is a withPull request or the image default mode is to
	// pull-image-on-create(true) and no-pull was not set we pull the image when
	// they ask for a create as a helper on the cli to reduce extra steps. As a
	// reminder if the image is already in cache only the manifest will be pulled
	// down to verify.
	if withPull {
		// Try to pull the image before container creation
		ann := ctrConfig.GetImage().GetAnnotations()
		resp, err := c.PullImageWithSandbox(ctx, image, nil, nil, ann)
		if err != nil {
			return "", errors.Wrap(err, "PullImageWithSandbox")
		}
		log.Infof("Pull image %s", resp.String())
	}

	log.Infof("CreateContainerRequest: %v", req)
	r, err := c.getRuntimeClient().CreateContainer(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "CreateContainer with: %s", req)
	}
	return r.GetContainerId(), nil
}

// StartContainer sends a StartContainerRequest to the server, and parses
// the returned StartContainerResponse.
func (c crictl) StartContainer(ctx context.Context, id string) error {
	if id == "" {
		return errors.Error("Id can't be empty")
	}
	if _, err := c.getRuntimeClient().StartContainer(ctx, &runtimeapi.StartContainerRequest{
		ContainerId: id,
	}); err != nil {
		return errors.Wrapf(err, "StartContainer %s", id)
	}
	return nil
}

type RunContainersResponse struct {
	PodId        string   `json:"pod_id"`
	ContainerIds []string `json:"container_ids"`
}

// RunContainers starts containers in the provided pod sandbox
func (c crictl) RunContainers(
	ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, containerConfigs []*runtimeapi.ContainerConfig, runtimeHandler string) (*RunContainersResponse, error) {
	// Create the pod
	podId, err := c.RunPodSandbox(ctx, podConfig, runtimeHandler)
	if err != nil {
		return nil, errors.Wrap(err, "RunPodSandbox")
	}
	ret := &RunContainersResponse{
		PodId:        podId,
		ContainerIds: make([]string, 0),
	}
	// Create the containers
	for idx, ctr := range containerConfigs {
		// Create the container
		ctrId, err := c.CreateContainer(ctx, podId, podConfig, ctr, true)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateContainer %d", idx)
		}
		// Start the container
		if err := c.StartContainer(ctx, ctrId); err != nil {
			return nil, errors.Wrapf(err, "StarContainer %d", idx)
		}
		ret.ContainerIds = append(ret.ContainerIds, ctrId)
	}
	return ret, nil
}

func main() {
	//ctl, err := NewCRICtl("unix:///run/containerd/containerd.sock", 3*time.Second)
	ctl, err := NewCRICtl("unix:///var/run/yunion/containerd/containerd.sock", 3*time.Second)
	if err != nil {
		log.Fatalf("NewCRICtl", err)
	}
	ctx := context.Background()
	imgs, err := ctl.ListImages(ctx, nil)
	if err != nil {
		log.Fatalf("ListImages: %v", err)
	}
	for _, img := range imgs {
		log.Infof("get img: %s", img.String())
	}

	// create container
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      "run-pod-test",
			Uid:       "hdishd83dja3sbnduwk28bcs",
			Namespace: "yunion",
			Attempt:   0,
		},
		Hostname:     "run-pod-test",
		LogDirectory: "",
		DnsConfig:    nil,
		PortMappings: nil,
		Labels:       nil,
		Annotations:  nil,
		Linux:        nil,
		Windows:      nil,
	}
	ctrCfgs := []*runtimeapi.ContainerConfig{
		{
			Metadata: &runtimeapi.ContainerMetadata{
				Name: "contrainer-logger",
			},
			Image: &runtimeapi.ImageSpec{
				Image: "registry.cn-beijing.aliyuncs.com/yunionio/logger:v3.10.4",
			},
			Linux: &runtimeapi.LinuxContainerConfig{
				//SecurityContext: &runtimeapi.LinuxContainerSecurityContext{
				//	Privileged: true,
				//},
			},
		},
	}
	resp, err := ctl.RunContainers(ctx, podCfg, ctrCfgs, "")
	if err != nil {
		log.Fatalf("RunContainers: %v", err)
	}
	log.Infof("RunContainers: %s", jsonutils.Marshal(resp))
}
