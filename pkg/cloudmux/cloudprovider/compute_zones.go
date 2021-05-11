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
	_ IZone     = new(zoneAdapter)
	_ IZoneImpl = new(ZoneBaseImpl)
)

type IZone interface {
	List(opt *ZoneListOpt) ([]cloudprovider.ICloudZone, error)
	Get(id string) (cloudprovider.ICloudZone, error)
}

type IZoneImpl interface {
	List(opt *ZoneListOpt) (interface{}, error)
	Get(id string) (cloudprovider.ICloudZone, error)
}

type zoneAdapter struct {
	impl IZoneImpl
}

func NewZoneAdapter(impl IZoneImpl) IZone {
	return &zoneAdapter{
		impl: impl,
	}
}

type ZoneListOpt struct {
	Region string
}

func (a *zoneAdapter) List(opt *ZoneListOpt) ([]cloudprovider.ICloudZone, error) {
	zones, err := a.impl.List(opt)
	if err != nil {
		return nil, err
	}
	iZones := []cloudprovider.ICloudZone{}
	if err := ConvertInterfaceSlice(zones, &iZones); err != nil {
		return nil, err
	}
	return iZones, nil
}

func (a *zoneAdapter) Get(id string) (cloudprovider.ICloudZone, error) {
	return a.impl.Get(id)
}

type ZoneBaseImpl struct {
	*resourceBaseImpl
}

func NewZoneBase(p IProvider) *ZoneBaseImpl {
	return &ZoneBaseImpl{
		resourceBaseImpl: newResourceBase(p),
	}
}

func (z *ZoneBaseImpl) List(_ *ZoneListOpt) (interface{}, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "Zones.List")
}

func (z *ZoneBaseImpl) Get(id string) (cloudprovider.ICloudZone, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "Zones.Get")
}
