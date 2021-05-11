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

package aws

import (
	"yunion.io/x/pkg/errors"

	muxp "yunion.io/x/onecloud/pkg/cloudmux/cloudprovider"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
)

type instanceImpl struct {
	*muxp.InstanceBaseImpl

	provider *sAWSProvider
}

func newInstances(p *sAWSProvider) muxp.IInstance {
	i := &instanceImpl{
		InstanceBaseImpl: muxp.NewInstanceBase(p),
		provider:         p,
	}

	return muxp.NewInstanceAdapter(i)
}

func (i *instanceImpl) getClient(regionId string) (*aws.SRegion, error) {
	if regionId == "" {
		regionId = i.provider.defaultRegionId
	}
	cli := i.provider.getClient()
	return cli.GetRegion(regionId)
}

func (i *instanceImpl) withClient(callF func(*aws.SRegion) (interface{}, error)) (interface{}, error) {
	cli, err := i.getClient("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion client %s", i.provider.defaultRegionId)
	}
	return callF(cli)
}

func (i *instanceImpl) List(o *muxp.InstanceListOpt) (interface{}, error) {
	return i.withClient(func(cli *aws.SRegion) (interface{}, error) {
		instances, _, err := cli.GetInstances(o.Zone, nil, o.Offset, o.Limit)
		if err != nil {
			return nil, err
		}
		return instances, nil
	})
}

func (i *instanceImpl) Get(id string) (cloudprovider.ICloudVM, error) {
	ret, err := i.withClient(func(cli *aws.SRegion) (interface{}, error) {
		return cli.GetInstance(id)
	})
	if err != nil {
		return nil, err
	}
	return ret.(cloudprovider.ICloudVM), nil
}
