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
	_ IInstance     = new(instanceAdapter)
	_ IInstanceImpl = new(InstanceBaseImpl)
)

type IInstance interface {
	List(opt *InstanceListOpt) ([]cloudprovider.ICloudVM, error)
	Get(id string) (cloudprovider.ICloudVM, error)
	Create(input interface{}) (cloudprovider.ICloudVM, error)
}

type IInstanceImpl interface {
	List(opt *InstanceListOpt) (interface{}, error)
	Get(id string) (cloudprovider.ICloudVM, error)
	Create(input interface{}) (cloudprovider.ICloudVM, error)
}

type instanceAdapter struct {
	impl IInstanceImpl
}

func NewInstanceAdapter(impl IInstanceImpl) IInstance {
	return &instanceAdapter{
		impl: impl,
	}
}

type InstanceListOpt struct {
	Region string
	Zone   string
	Limit  int
	Offset int
}

func (i *instanceAdapter) List(opt *InstanceListOpt) ([]cloudprovider.ICloudVM, error) {
	instances, err := i.impl.List(opt)
	if err != nil {
		return nil, err
	}
	ivms := []cloudprovider.ICloudVM{}
	if err := ConvertInterfaceSlice(instances, &ivms); err != nil {
		return nil, err
	}
	return ivms, nil
}

func (i *instanceAdapter) Get(id string) (cloudprovider.ICloudVM, error) {
	vm, err := i.impl.Get(id)
	if err != nil {
		return nil, err
	}
	return vm.(cloudprovider.ICloudVM), nil
}

func (i *instanceAdapter) Create(input interface{}) (cloudprovider.ICloudVM, error) {
	vm, err := i.impl.Create(input)
	if err != nil {
		return nil, err
	}
	return vm.(cloudprovider.ICloudVM), nil
}

type InstanceBaseImpl struct {
	*resourceBaseImpl
}

func NewInstanceBase(p IProvider) *InstanceBaseImpl {
	return &InstanceBaseImpl{
		resourceBaseImpl: newResourceBase(p),
	}
}

func (_ InstanceBaseImpl) List(opt *InstanceListOpt) (interface{}, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "Instances.List")
}

func (_ InstanceBaseImpl) Get(id string) (cloudprovider.ICloudVM, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "Instances.Get")
}

func (_ InstanceBaseImpl) Create(input interface{}) (cloudprovider.ICloudVM, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "Instances.Create")
}
