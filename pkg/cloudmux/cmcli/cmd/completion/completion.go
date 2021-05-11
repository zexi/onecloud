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

package completion

import (
	"os"

	"github.com/spf13/cobra"

	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/util/templates"
)

var (
	getLong = templates.LongDesc(`
		To load completions:

		Bash:

		  $ source <(cmcli completion bash)

		Zsh:

		  $ source <(cmcli completion zsh)

		fish:

		  $ cmcli completion fish | source`)
)

func NewCmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish]",
		Short:                 "Generate completion script",
		Long:                  getLong,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			}
		},
	}
	return cmd
}
