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

type zoneImpl struct {
	*muxp.ZoneBaseImpl
}

func newZones(p *sAliyunProvider) muxp.IZone {
	z := &zoneImpl{
		ZoneBaseImpl: muxp.NewZoneBase(p),
	}

	return muxp.NewZoneAdapter(z)
}

func (i *zoneImpl) List(opt *muxp.ZoneListOpt) (interface{}, error) {
	return RegionClientCall(i, opt.Region, func(cli *aliyun.SRegion) (interface{}, error) {
		return cli.GetIZones()
	})
}

func (i *zoneImpl) Get(id string) (cloudprovider.ICloudZone, error) {
	return nil, nil
}
