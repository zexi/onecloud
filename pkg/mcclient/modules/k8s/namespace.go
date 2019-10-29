// Copyright 2019 Yunion
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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Namespaces     *NamespaceManager
	LimitRanges    *LimitRangeManager
	ResourceQuotas *ResourceQuotaManager
)

type NamespaceManager struct {
	*MetaResourceManager
	statusGetter
}

type LimitRangeManager struct {
	*NamespaceResourceManager
}

type ResourceQuotaManager struct {
	*NamespaceResourceManager
}

func init() {
	Namespaces = &NamespaceManager{
		MetaResourceManager: NewMetaResourceManager("namespace", "namespaces", NewNameCols("Status"), NewClusterCols()),
		statusGetter:        getStatus,
	}

	LimitRanges = &LimitRangeManager{
		NamespaceResourceManager: NewNamespaceResourceManager(
			"limitrange", "limitranges",
			NewColumns(), NewColumns(),
		),
	}

	ResourceQuotas = &ResourceQuotaManager{
		NamespaceResourceManager: NewNamespaceResourceManager(
			"resourcequota", "resourcequotas",
			NewColumns(), NewColumns(),
		),
	}

	modules.Register(Namespaces)
	modules.Register(LimitRanges)
	modules.Register(ResourceQuotas)
}
