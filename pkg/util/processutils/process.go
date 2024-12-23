package processutils

import (
	"github.com/shirou/gopsutil/process"

	"yunion.io/x/pkg/errors"
)

func GetProcessChildrenWithSelf(pid int32) ([]int32, error) {
	return GetProcessChildren(pid, true)
}

func GetProcessChildren(pid int32, withSelf bool) ([]int32, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, errors.Wrapf(err, "NewProcess by %d", pid)
	}
	result := []int32{}
	if withSelf {
		result = append(result, pid)
	}
	children, err := p.Children()
	if err != nil {
		if errors.Cause(err) == process.ErrorNoChildren {
			return result, nil
		}
		return nil, errors.Wrapf(err, "Children by %d", pid)
	}

	for _, child := range children {
		childIds, err := GetProcessChildren(child.Pid, true)
		if err != nil {
			return nil, errors.Wrapf(err, "GetProcessChildren by %d", child.Pid)
		}
		result = append(result, childIds...)
	}
	return result, nil
}
