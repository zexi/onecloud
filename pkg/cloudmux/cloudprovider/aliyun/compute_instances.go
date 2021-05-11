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

package aliyun

import (
	muxp "yunion.io/x/onecloud/pkg/cloudmux/cloudprovider"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
)

type instanceImpl struct {
	*muxp.InstanceBaseImpl
}

func newInstances(p *sAliyunProvider) muxp.IInstance {
	i := &instanceImpl{
		InstanceBaseImpl: muxp.NewInstanceBase(p),
	}

	return muxp.NewInstanceAdapter(i)
}

func (i *instanceImpl) List(opt *muxp.InstanceListOpt) (interface{}, error) {
	return RegionClientCall(i, opt.Region, func(cli *aliyun.SRegion) (interface{}, error) {
		/*
		 * for {
		 *     parts, total, err := region.GetInstances(zone, nil, offset, limit)
		 *     if err != nil {
		 *         return nil, err
		 *     }
		 *     vms = append(vms, parts...)
		 *     if len(vms) >= total {
		 *         break
		 *     }
		 * }
		 */
		parts, _, err := cli.GetInstances(opt.Zone, nil, opt.Offset, opt.Limit)
		if err != nil {
			return nil, err
		}

		return parts, nil
	})
}

func (i *instanceImpl) Get(id string) (cloudprovider.ICloudVM, error) {
	ret, err := RegionClientCall(i, "", func(cli *aliyun.SRegion) (interface{}, error) {
		return cli.GetInstance(id)
	})
	if err != nil {
		return nil, err
	}

	return ret.(cloudprovider.ICloudVM), nil
}
