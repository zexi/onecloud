package guestman

import (
	"context"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type GuestRuntimeInstance interface {
	HomeDir() string
	GetDesc() *desc.SGuestDesc
	SetDesc(guestDesc *desc.SGuestDesc)
	GetSourceDesc() *desc.SGuestDesc
	SetSourceDesc(guestDesc *desc.SGuestDesc)
	GetDescFilePath() string
	DeployFs(ctx context.Context, userCred mcclient.TokenCredential, deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error)
	GetSourceDescFilePath() string
	IsRunning() bool
}

type sBaseGuestInstance struct {
	Id      string
	manager *SGuestManager
	// runtime description, generate from source desc
	Desc *desc.SGuestDesc
	// source description, input from region
	SourceDesc *desc.SGuestDesc
	Hypervisor string
}

func newBaseGuestInstance(id string, manager *SGuestManager, hypervisor string) *sBaseGuestInstance {
	return &sBaseGuestInstance{
		Id:         id,
		manager:    manager,
		Hypervisor: hypervisor,
	}
}

func (b *sBaseGuestInstance) HomeDir() string {
	return path.Join(b.manager.ServersPath, b.Id)
}

func (s *sBaseGuestInstance) GetDescFilePath() string {
	return path.Join(s.HomeDir(), "desc")
}

func (s *sBaseGuestInstance) GetDesc() *desc.SGuestDesc {
	return s.Desc
}

func (s *sBaseGuestInstance) SetDesc(guestDesc *desc.SGuestDesc) {
	s.Desc = guestDesc
}

func (s *sBaseGuestInstance) GetSourceDesc() *desc.SGuestDesc {
	return s.SourceDesc
}

func (s *sBaseGuestInstance) SetSourceDesc(guestDesc *desc.SGuestDesc) {
	s.SourceDesc = guestDesc
}

func (s *sBaseGuestInstance) GetSourceDescFilePath() string {
	return path.Join(s.HomeDir(), "source-desc")
}

type GuestRuntimeManager struct {
}

func NewGuestRuntimeManager() *GuestRuntimeManager {
	return new(GuestRuntimeManager)
}

func (f *GuestRuntimeManager) NewRuntimeInstance(id string, manager *SGuestManager, hypervisor string) GuestRuntimeInstance {
	switch hypervisor {
	case computeapi.HYPERVISOR_KVM, "":
		return NewKVMGuestInstance(id, manager)
	case computeapi.HYPERVISOR_POD:
		return newPodGuestInstance(id, manager)
	}
	log.Fatalf("Invalid hypervisor for runtime: %q", hypervisor)
	return nil
}

func (f *GuestRuntimeManager) PrepareDir(s GuestRuntimeInstance) error {
	output, err := procutils.NewCommand("mkdir", "-p", s.HomeDir()).Output()
	if err != nil {
		return errors.Wrapf(err, "mkdir %s failed: %s", s.HomeDir(), output)
	}
	return nil
}

func (f *GuestRuntimeManager) CreateFromDesc(s GuestRuntimeInstance, desc *desc.SGuestDesc) error {
	if err := f.PrepareDir(s); err != nil {
		return errors.Errorf("Failed to create server dir %s", desc.Uuid)
	}
	return SaveDesc(s, desc)
}
func SaveDesc(s GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error {
	s.SetSourceDesc(guestDesc)
	// fill in ovn vpc nic bridge field
	for _, nic := range s.GetSourceDesc().Nics {
		if nic.Bridge == "" {
			nic.Bridge = getNicBridge(nic)
		}
	}

	if err := fileutils2.FilePutContents(
		s.GetSourceDescFilePath(), jsonutils.Marshal(s.GetSourceDesc()).String(), false,
	); err != nil {
		log.Errorf("save source desc failed %s", err)
		return errors.Wrap(err, "source save desc")
	}

	if !s.IsRunning() { // if guest not running, sync live desc
		liveDesc := new(desc.SGuestDesc)
		if err := jsonutils.Marshal(s.GetSourceDesc()).Unmarshal(liveDesc); err != nil {
			return errors.Wrap(err, "unmarshal live desc")
		}
		return SaveLiveDesc(s, liveDesc)
	}
	return nil
}

func SaveLiveDesc(s GuestRuntimeInstance, guestDesc *desc.SGuestDesc) error {
	s.SetDesc(guestDesc)

	defaultGwCnt := 0
	defNics := netutils2.SNicInfoList{}
	// fill in ovn vpc nic bridge field
	for _, nic := range s.GetDesc().Nics {
		if nic.Bridge == "" {
			nic.Bridge = getNicBridge(nic)
		}
		if nic.IsDefault {
			defaultGwCnt++
		}
		defNics = defNics.Add(nic.Ip, nic.Mac, nic.Gateway)
	}

	// there should 1 and only 1 default gateway
	if defaultGwCnt != 1 {
		// fix is_default
		_, defIndex := defNics.FindDefaultNicMac()
		for i := range s.GetDesc().Nics {
			if i == defIndex {
				s.GetDesc().Nics[i].IsDefault = true
			} else {
				s.GetDesc().Nics[i].IsDefault = false
			}
		}
	}

	if err := fileutils2.FilePutContents(
		s.GetDescFilePath(), jsonutils.Marshal(s.GetDesc()).String(), false,
	); err != nil {
		log.Errorf("save desc failed %s", err)
		return errors.Wrap(err, "save desc")
	}
	return nil
}
