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
	"yunion.io/x/onecloud/pkg/multicloud/aws/provider"
)

type SAWSFactory struct {
	// CloudEnv
	CloudEnv string `json:"access_url"`
	// Access key
	AccessKey string `json:"access_key"`
	// Secret
	Secret string `json:"secret"`
	// RegionId
	RegionId string `json:"region_id"`

	// AccountId
	//  AccountId string `json:"account_id"`

	// debug
	debug bool
}

func (c *SAWSFactory) SetDebug(debug bool) muxp.IProviderFactory {
	c.debug = debug
	return c
}

func (c *SAWSFactory) IsInit() bool {
	return c.AccessKey != "" && c.Secret != ""
}

func (c *SAWSFactory) GetName() string {
	return "aws"
}

func (c *SAWSFactory) GetProvider() (muxp.IProvider, error) {
	return newAWSProvider(c)
}

type sAWSProvider struct {
	*muxp.SProviderBase

	debug           bool
	defaultRegionId string
}

func newAWSProvider(f *SAWSFactory) (muxp.IProvider, error) {
	factory, err := cloudprovider.GetProviderFactory(aws.CLOUD_PROVIDER_AWS)
	if err != nil {
		return nil, errors.Wrap(err, "get AWS cloudprovider factory")
	}

	baseP, err := factory.GetProvider(cloudprovider.ProviderConfig{
		URL:     f.CloudEnv,
		Vendor:  aws.CLOUD_PROVIDER_AWS,
		Account: f.AccessKey,
		Secret:  f.Secret,
	})
	if err != nil {
		return nil, err
	}

	p := &sAWSProvider{
		SProviderBase: muxp.NewProviderBase(baseP),

		debug:           f.debug,
		defaultRegionId: f.RegionId,
	}

	p.GetComputeImpl().SetInstances(newInstances(p))

	return p, nil
}

func (p *sAWSProvider) getClient() *aws.SAwsClient {
	cli := p.ICloudProvider.(*provider.SAwsProvider).GetClient()
	cli.Debug(p.debug)
	return cli
}
