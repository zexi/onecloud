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

package instances

import (
	"github.com/spf13/cobra"

	"yunion.io/x/onecloud/pkg/cloudmux/cloudprovider"
	"yunion.io/x/onecloud/pkg/cloudmux/cmcli/util/cmd"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

type ListOptions struct {
	// Zone is cloud zone
	Zone string
	// IDs of instances to show
	IDs []string
	// Limit is page size
	Limit int
	// Offset is page offset
	Offset int
}

func NewListOptions() *ListOptions {
	return &ListOptions{
		IDs: make([]string, 0),
	}
}

func NewCmdInstancesList(f cmd.Factory) *cobra.Command {
	o := NewListOptions()

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual machine instances",
		Run: f.RunWithProvider(func(provider cloudprovider.IProvider, _ []string) error {
			instances, err := provider.Compute().Instances().List(
				&cloudprovider.InstanceListOpt{
					Zone:   o.Zone,
					Limit:  o.Limit,
					Offset: o.Offset,
				},
			)
			if err != nil {
				return err
			}
			printutils.PrintInterfaceList(instances, len(instances), o.Offset, o.Limit, nil)
			return err
		}),
	}

	cmd.Flags().StringVar(&o.Zone, "zone", o.Zone, "If provided, only resources from the given zones are queried.")
	cmd.Flags().StringSliceVar(&o.IDs, "id", o.IDs, "Filter resources by provided ID")
	cmd.Flags().IntVar(&o.Limit, "limit", o.Limit, "The maximum number of results.")
	cmd.Flags().IntVar(&o.Offset, "offset", o.Offset, "Page offset")
	return cmd
}
