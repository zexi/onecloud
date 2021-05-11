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
	"testing"

	"github.com/stretchr/testify/assert"
)

type stringer interface {
	String() string
	Set(string)
}

type stringImpl struct {
	elem string
}

func (s stringImpl) String() string {
	return s.elem
}

func (s *stringImpl) Set(elem string) {
	s.elem = elem
}

func TestConvertInterfaceSlice(t *testing.T) {
	newElem := func(elem string) stringImpl {
		return stringImpl{elem}
	}

	newElemPtr := func(elem string) *stringImpl {
		return &stringImpl{elem}
	}

	inputs := []stringImpl{
		newElem("1"), newElem("2"),
	}

	inputPtrs := []*stringImpl{
		newElemPtr("1 ptr"), newElemPtr("2 ptr"),
	}

	outputs := []stringer{}
	outputPtrs := []stringer{}

	if err := ConvertInterfaceSlice(inputs, &outputs); err != nil {
		t.Error(err)
	}

	if err := ConvertInterfaceSlice(inputPtrs, &outputPtrs); err != nil {
		t.Error(err)
	}

	outputs[0].Set("new 1")
	assert.Equal(t, "new 1", inputs[0].String())

	outputPtrs[0].Set("new 1 ptr")
	assert.Equal(t, "new 1 ptr", inputPtrs[0].String())
}
