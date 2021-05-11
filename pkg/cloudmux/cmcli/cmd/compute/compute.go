// Copyright 2021 Yunion
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
	"github.com/spf13/cobra"

	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/cmd/compute/instances"
	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/cmd/compute/regions"
	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/cmd/compute/zones"
	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/util/cmd"
)

func NewCmdCompute(f cmd.Factory) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "compute",
		Short: "Manipulate Compute Engine releated resources",
		Long:  "The compute command group lets you create, configure, and manipulate Compute Engine virtual machine(VM) instances.",
	}

	cmds.AddCommand(
		regions.NewCmdRegions(f),
		instances.NewCmdInstances(f),
		zones.NewCmdZones(f),
	)

	return cmds
}
