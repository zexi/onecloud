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
	"yunion.io/x/pkg/errors"

	muxp "yunion.io/x/onecloud/pkg/cloudmux/cloudprovider"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun/provider"
)

var (
	_ muxp.IProvider = new(sAliyunProvider)
)

type SAliyunFactory struct {
	// CloudEnvironment is aliyun cloud environment, choices 'InternationalCloud' or 'FinanceCloud'
	CloudEnvironment string `json:"cloud_environment"`
	// Access key
	AccessKey string `json:"access_key"`
	// Secret
	Secret string `json:"secret"`
	// RegionId
	RegionId string `json:"region_id"`

	debug bool
}

func (c *SAliyunFactory) IsInit() bool {
	return c.AccessKey != "" && c.Secret != ""
}

func (c *SAliyunFactory) GetName() string {
	return "aliyun"
}

func (c *SAliyunFactory) SetDebug(debug bool) muxp.IProviderFactory {
	c.debug = debug
	return c
}

func (c *SAliyunFactory) GetProvider() (muxp.IProvider, error) {
	return newAliyunProvider(c)
}

type sAliyunProvider struct {
	*muxp.SProviderBase

	defaultRegion string
	debug         bool
}

func newAliyunProvider(f *SAliyunFactory) (muxp.IProvider, error) {
	factory, err := cloudprovider.GetProviderFactory(aliyun.CLOUD_PROVIDER_ALIYUN)
	if err != nil {
		return nil, errors.Wrap(err, "get aliyun cloudprovider factory")
	}

	baseP, err := factory.GetProvider(cloudprovider.ProviderConfig{
		Vendor:  aliyun.CLOUD_PROVIDER_ALIYUN,
		URL:     f.CloudEnvironment,
		Account: f.AccessKey,
		Secret:  f.Secret,
	})
	if err != nil {
		return nil, err
	}

	p := &sAliyunProvider{
		SProviderBase: muxp.NewProviderBase(baseP),
		defaultRegion: f.RegionId,
		debug:         f.debug,
	}

	p.GetComputeImpl().SetInstances(newInstances(p))
	p.GetComputeImpl().SetZones(newZones(p))

	return p, nil
}

func (p *sAliyunProvider) GetClient() (interface{}, error) {
	return p.getClient(), nil
}

func (p *sAliyunProvider) getClient() *aliyun.SAliyunClient {
	cli := p.ICloudProvider.(*provider.SAliyunProvider).GetClient()
	cli.Debug(p.debug)
	return cli
}

func (p *sAliyunProvider) GetRegionClient(regionId string) (interface{}, error) {
	if regionId == "" {
		regionId = p.defaultRegion
	}
	return p.getClient().GetRegion(regionId), nil
}
