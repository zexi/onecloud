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
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var (
	_ IProviderFactory = new(SProviderFactory)
)

type IProviderFactory interface {
	IsInit() bool
	SetDebug(bool) IProviderFactory
	GetName() string
	GetProvider() (IProvider, error)
}

type IProvider interface {
	GetCloudProvider() cloudprovider.ICloudProvider

	// GetClient get cloud API client
	GetClient() (interface{}, error)
	// GetRegionClient get cloud region level resource API client
	GetRegionClient(regionId string) (interface{}, error)

	Compute() ICompute
}

type IRegion interface {
	List() ([]cloudprovider.ICloudRegion, error)
}

type SProviderFactory struct {
	// providerMap cache cloudprovider initialized
	providerMap sync.Map

	currentFactory IProviderFactory
}

func NewProviderFactory() *SProviderFactory {
	config := &SProviderFactory{
		providerMap:    sync.Map{},
		currentFactory: nil,
	}

	return config
}

func (c *SProviderFactory) IsInit() bool {
	return c.currentFactory != nil
}

func (c *SProviderFactory) SetDebug(debug bool) IProviderFactory {
	c.currentFactory.SetDebug(debug)
	return c
}

func (c *SProviderFactory) GetName() string {
	return c.currentFactory.GetName()
}

func (c *SProviderFactory) GetProvider() (IProvider, error) {
	if c.currentFactory == nil {
		return nil, errors.Errorf("Not specify current provider")
	}

	providerI, find := c.providerMap.Load(c.GetName())
	if find {
		return providerI.(IProvider), nil
	}

	provider, err := c.currentFactory.GetProvider()
	if err != nil {
		return nil, errors.Wrapf(err, "Construct provider %s", c.GetName())
	}

	// cache provider in map
	c.providerMap.Store(c.GetName(), provider)

	return provider, nil
}

func (c *SProviderFactory) Use(pf IProviderFactory) *SProviderFactory {
	log.Debugf("Use provider %s", pf.GetName())
	c.currentFactory = pf
	return c
}
