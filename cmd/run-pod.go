package main

import (
	"context"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/pod"
)

func main() {
	//ctl, err := NewCRI("unix:///run/containerd/containerd.sock", 3*time.Second)
	ctl, err := pod.NewCRI("unix:///var/run/onecloud/containerd/containerd.sock", 3*time.Second)
	if err != nil {
		log.Fatalf("NewCRI: %v", err)
	}
	ctx := context.Background()
	imgs, err := ctl.ListImages(ctx, nil)
	if err != nil {
		log.Fatalf("ListImages: %v", err)
	}
	for _, img := range imgs {
		log.Infof("get img: %s", img.String())
	}

	ver, err := ctl.Version(context.Background())
	if err != nil {
		log.Fatalf("======get version: %v", err)
	}
	log.Infof("===get version: %s", ver.String())

	// create container
	podCfg := &runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{
			Name:      "test-pod5",
			Uid:       "e25e38ef-fe98-4993-8641-699cd0530fc0",
			Namespace: "27c9464ab54947328a29298761895be3",
		},
		Hostname:     "test-pod-5",
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
				Image: "nginx",
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
