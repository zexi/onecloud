// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compute

import (
	"fmt"

	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type PodCreateOptions struct {
	NAME        string `help:"Name of server pod" json:"-"`
	IMAGE       string `help:"Image of container" json:"image"`
	MEM         string `help:"Memory size MB" metavar:"MEM" json:"-"`
	VcpuCount   int    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	AllowDelete *bool  `help:"Unlock server to allow deleting" json:"-"`

	ServerCreateCommonConfig
}

func (o *PodCreateOptions) Params() (*computeapi.ServerCreateInput, error) {
	config, err := o.ServerCreateCommonConfig.Data()
	if err != nil {
		return nil, err
	}
	config.Hypervisor = computeapi.HYPERVISOR_POD

	params := &computeapi.ServerCreateInput{
		ServerConfigs: config,
		VcpuCount:     o.VcpuCount,
		Pod: &computeapi.PodCreateInput{
			Containers: []*computeapi.PodContainerCreateInput{
				{
					Image: o.IMAGE,
				},
			},
		},
	}
	if options.BoolV(o.AllowDelete) {
		disableDelete := false
		params.DisableDelete = &disableDelete
	}
	if regutils.MatchSize(o.MEM) {
		memSize, err := fileutils.GetSizeMb(o.MEM, 'M', 1024)
		if err != nil {
			return nil, err
		}
		params.VmemSize = memSize
	} else {
		return nil, fmt.Errorf("Invalid memory input: %q", o.MEM)
	}
	params.Name = o.NAME
	return params, nil
}
