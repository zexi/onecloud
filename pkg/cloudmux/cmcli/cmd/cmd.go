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

package cmd

import (
	"github.com/spf13/cobra"

	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/cmd/completion"
	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/cmd/compute"
	cmdutil "yunion.io/x/onecloud/pkg/cloudmux/cmcli/util/cmd"
)

func NewCmcliCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmcli",
		Short: "Controls the multi cloud resources",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	opt := cmdutil.NewCloudMuxOptions(cmd.PersistentFlags())

	cmd.AddCommand(compute.NewCmdCompute(opt))

	cmd.AddCommand(completion.NewCmdCompletion())

	return cmd
}
