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

package monitor

import (
	"fmt"
	"strings"
	"time"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertListOptions struct {
	options.BaseListOptions
}

type AlertShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

type AlertDeleteOptions struct {
	ID []string `help:"ID of alert to delete"`
}

type AlertTestRunOptions struct {
	ID    string `help:"ID of alert to delete"`
	Debug bool   `help:"Show more debug info"`
}

type AlertConditionOptions struct {
	REDUCER    string   `help:"Metric query reducer, e.g. 'avg'" choices:"avg|sum|min|max|count|last|median"`
	DATABASE   string `help:"Metric database, e.g. 'telegraf'"`
	METRIC     string   `help:"Query metric format <measurement>.<field>, e.g. 'cpu.cpu_usage'"`
	COMPARATOR string   `help:"Evaluator compare" choices:"gt|lt"`
	THRESHOLD  float64  `help:"Alert threshold"`
	Period     string   `help:"Query metric period e.g. '5m', '1h'" default:"5m"`
	Tag        []string `help:"Query tag, e.g. 'zone=zon0,name=vmname'"`
}

func (opt AlertConditionOptions) Params() (*monitor.AlertCondition, error) {
	cond := new(monitor.AlertCondition)
	cond.Type = "query"
	cond.Operator = "and"
	cond.Query = monitor.AlertQuery{
		From: opt.Period,
	}
	cond.Reducer = monitor.Condition{
		Type: opt.REDUCER,
	}
	cond.Evaluator = monitor.Condition{
		Type:   opt.COMPARATOR,
		Params: []float64{opt.THRESHOLD},
	}

	parts := strings.Split(opt.METRIC, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("metric %s is invalid format", opt.METRIC)
	}
	sels := make([]monitor.MetricQuerySelect, 0)
	sels = append(sels, monitor.NewMetricQuerySelect(monitor.MetricQueryPart{Type: "field", Params: []string{parts[1]}}))
	model := monitor.MetricQuery{
		Selects:     sels,
		Measurement: parts[0],
		Database: opt.DATABASE,
	}

	tags := make([]monitor.MetricQueryTag, 0)
	for _, tag := range opt.Tag {
		parts := strings.Split(tag, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format: %s", tag)
		}
		tags = append(tags, monitor.MetricQueryTag{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	model.Tags = tags
	cond.Query.Model = model

	return cond, nil
}

type AlertCreateOptions struct {
	AlertConditionOptions
	NAME      string `help:"Name of the alert"`
	Frequency string `help:"Alert execute frequency, e.g. '5m', '1h'"`
	Enabled   bool   `help:"Enable alert"`
	Level     string `help:"Alert level"`
}

func (opt AlertCreateOptions) Params() (*monitor.AlertCreateInput, error) {
	input := new(monitor.AlertCreateInput)
	input.Name = opt.NAME
	if opt.Frequency != "" {
		freq, err := time.ParseDuration(opt.Frequency)
		if err != nil {
			return nil, fmt.Errorf("Invalid frequency time format %s: %v", opt.Frequency, err)
		}
		f := int64(freq / time.Second)
		input.Frequency = f
	}
	input.Enabled = &opt.Enabled

	cond, err := opt.AlertConditionOptions.Params()
	if err != nil {
		return nil, err
	}

	input.Settings.Conditions = append(input.Settings.Conditions, *cond)

	return input, nil
}

type AlertUpdateOptions struct {
	ID        string `help:"ID or name of the alert"`
	Name      string `help:"Update alert name"`
	Frequency string `help:"Alert execute frequency, e.g. '5m', '1h'"`
}

func (opt AlertUpdateOptions) Params() (*monitor.AlertUpdateInput, error) {
	input := new(monitor.AlertUpdateInput)
	if opt.Name != "" {
		input.Name = opt.Name
	}
	if opt.Frequency != "" {
		freq, err := time.ParseDuration(opt.Frequency)
		if err != nil {
			return nil, fmt.Errorf("Invalid frequency time format %s: %v", opt.Frequency, err)
		}
		f := int64(freq / time.Second)
		input.Frequency = &f
	}
	return input, nil
}

type AlertNotificationAttachOptions struct {
	ALERT string `help:"ID or name of alert"`
	NOTIFICATION string `help:"ID or name of alert notification"`
	UsedBy string `help:"UsedBy annotation"`
}

type AlertNotificationListOptions struct {
	options.BaseListOptions
	Alert string `help:"ID or name of alert" short-token:"a"`
	Notification string `help:"ID or name of notification" short-token:"n"`
}

func (o AlertNotificationListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}
