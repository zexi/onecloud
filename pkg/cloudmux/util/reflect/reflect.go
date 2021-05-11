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

package reflect

import (
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func ConvertInterfaceSlice(inputs interface{}, targets interface{}) error {
	inputVal := reflect.ValueOf(inputs)
	targetPtrVal := reflect.ValueOf(targets)

	if inputVal.Kind() != reflect.Slice {
		return errors.Errorf("inputs args is not slice")
	}

	targetVal := reflect.Indirect(targetPtrVal)
	if targetVal.Kind() != reflect.Slice {
		return errors.Errorf("targets args kind %s is not slice", targetVal.Kind())
	}

	for i := 0; i < inputVal.Len(); i++ {
		itemV := inputVal.Index(i)

		log.Infof("%s, %T", itemV.Kind(), itemV.Addr().Interface())
		itemPtr := itemV
		if itemV.Kind() != reflect.Ptr && itemV.Kind() != reflect.Interface {
			itemPtr = reflect.ValueOf(itemV.Addr().Interface())
		}

		newTargets := reflect.Append(targetVal, itemPtr)
		targetVal.Set(newTargets)
	}

	return nil
}
