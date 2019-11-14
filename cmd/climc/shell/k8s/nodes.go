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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initK8sNode() {
	cmd := initK8sClusterResource("node", k8s.K8sNodes)
	cmdN := cmd.CommandNameFactory

	cordonCmd := NewCommand(
		&o.ResourceGetOptions{},
		cmdN("cordon"),
		"Set node unschedule",
		func(s *mcclient.ClientSession, args *o.ResourceGetOptions) error {
			params := args.Params()
			ret, err := k8s.K8sNodes.PerformAction(s, args.NAME, "cordon", params)
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		})

	uncordonCmd := NewCommand(
		&o.ResourceGetOptions{},
		cmdN("uncordon"),
		"Set node schedule",
		func(s *mcclient.ClientSession, args *o.ResourceGetOptions) error {
			params := args.Params()
			ret, err := k8s.K8sNodes.PerformAction(s, args.NAME, "uncordon", params)
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		})

	cmd.AddR(cordonCmd, uncordonCmd)
}
