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

package compute

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ContainerListOptions struct {
	options.BaseListOptions
	GuestId string `json:"guest_id" help:"guest(pod) id or name"`
}

func (o *ContainerListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type ContainerDeleteOptions struct {
	ServerIdsOptions
}

type ContainerCreateOptions struct {
	PODID      string   `help:"Name or id of server pod" json:"-"`
	NAME       string   `help:"Name of container" json:"-"`
	IMAGE      string   `help:"Image of container" json:"image"`
	Command    []string `help:"Command to execute (i.e., entrypoint for docker)" json:"command"`
	Args       []string `help:"Args for the Command (i.e. command for docker)" json:"args"`
	WorkingDir string   `help:"Current working directory of the command" json:"working_dir"`
	Env        []string `help:"List of environment variable to set in the container and format is: <key>=<value>"`
}

func (o *ContainerCreateOptions) Params() (jsonutils.JSONObject, error) {
	req := computeapi.ContainerCreateInput{
		GuestId: o.PODID,
		Spec: computeapi.ContainerSpec{
			Image:      o.IMAGE,
			Command:    o.Command,
			Args:       o.Args,
			WorkingDir: o.WorkingDir,
			Envs:       make([]*computeapi.ContainerKeyValue, 0),
		},
	}
	req.Name = o.NAME
	for _, env := range o.Env {
		e, err := parseContainerEnv(env)
		if err != nil {
			return nil, errors.Wrapf(err, "parseContainerEnv %s", env)
		}
		req.Spec.Envs = append(req.Spec.Envs, e)
	}
	return jsonutils.Marshal(req), nil
}

func parseContainerEnv(env string) (*computeapi.ContainerKeyValue, error) {
	kv := strings.Split(env, "=")
	if len(kv) != 2 {
		return nil, errors.Errorf("invalid env: %q", env)
	}
	return &computeapi.ContainerKeyValue{
		Key:   kv[0],
		Value: kv[1],
	}, nil
}
