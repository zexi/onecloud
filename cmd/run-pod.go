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
	ctl, err := pod.NewCRI("unix:///var/run/yunion/containerd/containerd.sock", 3*time.Second)
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
