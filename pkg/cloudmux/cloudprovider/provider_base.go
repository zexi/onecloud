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

package cloudprovider

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var (
	_ IProvider = new(SProviderBase)
)

type SProviderBase struct {
	cloudprovider.ICloudProvider

	compute *ComputeImpl
}

func NewProviderBase(p cloudprovider.ICloudProvider) *SProviderBase {
	bp := &SProviderBase{
		ICloudProvider: p,
	}

	bp.compute = NewComputeImpl(bp)

	return bp
}

func (p *SProviderBase) GetCloudProvider() cloudprovider.ICloudProvider {
	return p.ICloudProvider
}

func (p *SProviderBase) GetClient() (interface{}, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "GetClient must implement")
}

func (p *SProviderBase) GetRegionClient(regionId string) (interface{}, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "GetRegionClient must implement")
}

func (p *SProviderBase) GetComputeImpl() *ComputeImpl {
	return p.compute
}

func (p *SProviderBase) Compute() ICompute {
	return p.compute
}

type IResourceImpl interface {
	GetProvider() IProvider
}

type resourceBaseImpl struct {
	provider IProvider
}

func newResourceBase(p IProvider) *resourceBaseImpl {
	return &resourceBaseImpl{
		provider: p,
	}
}

func (i resourceBaseImpl) GetProvider() IProvider {
	return i.provider
}

func (i resourceBaseImpl) GetCloudProvider() cloudprovider.ICloudProvider {
	return i.GetProvider().GetCloudProvider()
}

type clientCallF func(client interface{}) (interface{}, error)

func ClientCall(i IResourceImpl, callF clientCallF) (interface{}, error) {
	cli, err := i.GetProvider().GetClient()
	if err != nil {
		return nil, errors.Wrap(err, "Get provider API client")
	}

	return callF(cli)
}

func RegionClientCall(i IResourceImpl, regionId string, callF clientCallF) (interface{}, error) {
	cli, err := i.GetProvider().GetRegionClient(regionId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get provider Region level resource API client")
	}

	return callF(cli)
}
