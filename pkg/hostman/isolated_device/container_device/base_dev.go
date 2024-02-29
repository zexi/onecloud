package container_device

import "yunion.io/x/onecloud/pkg/hostman/isolated_device"

type BaseDevice struct {
	*isolated_device.SBaseDevice
}

func NewBaseDevice(dev *isolated_device.PCIDevice, devType isolated_device.ContainerDeviceType) *BaseDevice {
	return &BaseDevice{
		SBaseDevice: isolated_device.NewBaseDevice(dev, string(devType)),
	}
}

func (b BaseDevice) GetVGACmd() string {
	return ""
}

func (b BaseDevice) GetCPUCmd() string {
	return ""
}

func (b BaseDevice) GetQemuId() string {
	return ""
}

func (c BaseDevice) CustomProbe(idx int) error {
	return nil
}
