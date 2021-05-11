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

var (
	_ ICompute = new(ComputeImpl)
)

// ICompute is compute service interface
type ICompute interface {
	Regions() IRegion
	Instances() IInstance
	Zones() IZone
}

type ComputeImpl struct {
	provider IProvider

	regions   IRegion
	instances IInstance
	zones     IZone
}

func NewComputeImpl(p IProvider) *ComputeImpl {
	c := &ComputeImpl{
		provider: p,
	}

	c.regions = NewRegionBase(c.provider)
	c.instances = NewInstanceAdapter(NewInstanceBase(c.provider))
	c.zones = NewZoneAdapter(NewZoneBase(c.provider))

	return c
}

func (c *ComputeImpl) SetRegions(regions IRegion) *ComputeImpl {
	c.regions = regions
	return c
}

func (c *ComputeImpl) SetInstances(instances IInstance) *ComputeImpl {
	c.instances = instances
	return c
}

func (c *ComputeImpl) SetZones(zones IZone) *ComputeImpl {
	c.zones = zones
	return c
}

func (c *ComputeImpl) Regions() IRegion {
	return c.regions
}

func (c *ComputeImpl) Instances() IInstance {
	return c.instances
}

func (c *ComputeImpl) Zones() IZone {
	return c.zones
}
