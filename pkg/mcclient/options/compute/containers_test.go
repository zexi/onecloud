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
	"reflect"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func Test_parseContainerVolumeMount(t *testing.T) {
	index0 := 0
	tests := []struct {
		args    string
		want    *computeapi.ContainerVolumeMount
		wantErr bool
	}{
		{
			args: "readonly=true,mount_path=/data,disk_index=0",
			want: &computeapi.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/data",
				Disk:      &computeapi.ContainerVolumeMountDisk{Index: &index0},
			},
		},
		{
			args: "readonly=True,mount_path=/test",
			want: &computeapi.ContainerVolumeMount{
				ReadOnly:  true,
				MountPath: "/test",
			},
		},
		{
			args:    "vm1,readonly=True,mount_path=/test",
			want:    nil,
			wantErr: true,
		},
		{
			args:    "readonly=True,mount_path=/test,disk_index=one",
			want:    nil,
			wantErr: true,
		},
		{
			args:    "disk_id=abc,mount_path=/data",
			wantErr: false,
			want: &computeapi.ContainerVolumeMount{
				Disk: &computeapi.ContainerVolumeMountDisk{
					Id: "abc",
				},
				MountPath: "/data",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := parseContainerVolumeMount(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContainerVolumeMount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseContainerVolumeMount() got = %v, want %v", got, tt.want)
			}
		})
	}
}
