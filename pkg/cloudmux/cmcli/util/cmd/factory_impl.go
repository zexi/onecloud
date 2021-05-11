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

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudmux/cloudprovider"
	"yunion.io/x/onecloud/pkg/cloudmux/cloudprovider/aliyun"
	"yunion.io/x/onecloud/pkg/cloudmux/cloudprovider/aws"
)

var (
	_ Factory = new(CloudMuxOptions)
)

type CloudMuxOptions struct {
	factoryList []cloudprovider.IProviderFactory

	Debug  bool                   `json:"debug"`
	Aliyun *aliyun.SAliyunFactory `json:"aliyun"`
	AWS    *aws.SAWSFactory       `json:"aws"`
}

func NewCloudMuxOptions(fs *pflag.FlagSet) *CloudMuxOptions {
	opt := &CloudMuxOptions{
		factoryList: make([]cloudprovider.IProviderFactory, 0),
	}

	fs.BoolVarP(&opt.Debug, "debug", "d", false, "Debug trigger")

	opt.initAliyunFlag(fs).
		initAWSFlag(fs)

	return opt
}

func (opt *CloudMuxOptions) addFactory(fs ...cloudprovider.IProviderFactory) *CloudMuxOptions {
	opt.factoryList = append(opt.factoryList, fs...)
	return opt
}

func (opt *CloudMuxOptions) initAliyunFlag(fs *pflag.FlagSet) *CloudMuxOptions {
	// Aliyun config options
	opt.Aliyun = &aliyun.SAliyunFactory{}

	fs.StringVar(&opt.Aliyun.CloudEnvironment, "aliyun-cloud-environment", "InternationalCloud", "Aliyun cloud environment, choices InternationalCloud or FinanceCloud")
	fs.StringVar(&opt.Aliyun.AccessKey, "aliyun-access-key", "", "Aliyun auth access_key")
	fs.StringVar(&opt.Aliyun.Secret, "aliyun-secret", "", "Aliyun auth secret")
	fs.StringVar(&opt.Aliyun.RegionId, "aliyun-region", "cn-hangzhou", "Aliyun region id")

	return opt.addFactory(opt.Aliyun)
}

func (opt *CloudMuxOptions) initAWSFlag(fs *pflag.FlagSet) *CloudMuxOptions {
	opt.AWS = &aws.SAWSFactory{}

	// AWS config options
	fs.StringVar(&opt.AWS.CloudEnv, "aws-cloud-env", "InternationalCloud", "AWS cloud environment, choices InternationalCloud or ChinaCloud")
	fs.StringVar(&opt.AWS.AccessKey, "aws-access-key", "", "AWS auth access_key")
	fs.StringVar(&opt.AWS.Secret, "aws-secret", "", "AWS auth secret")
	fs.StringVar(&opt.AWS.RegionId, "aws-region", "", "AWS region id")

	return opt.addFactory(opt.AWS)
}

func (opt *CloudMuxOptions) GetProvider() (cloudprovider.IProvider, error) {
	var initFactory cloudprovider.IProviderFactory

	for _, f := range opt.factoryList {
		if f.IsInit() {
			initFactory = f
			break
		}
	}
	if initFactory == nil {
		return nil, errors.Errorf("Not found provider initialized")
	}

	return cloudprovider.NewProviderFactory().
		Use(initFactory).SetDebug(opt.Debug).GetProvider()
}

func (opt *CloudMuxOptions) RunWithProvider(f RunWithProviderFunc) CobraRunFunc {
	return func(_ *cobra.Command, args []string) {
		provider, err := opt.GetProvider()
		if err != nil {
			log.Fatalf("Get provider: %v", err)
		}

		if err := f(provider, args); err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
	}
}
