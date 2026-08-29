package ps

import (
	"errors"
	"fmt"
	"github.com/iyear/pure-live-core/pkg/util"
	"os"
	"testing"
	"time"
)

func TestGetSysMem(t *testing.T) {
	m, err := GetSysMem()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	fmt.Println(m)
}

func TestGetSelfMem(t *testing.T) {
	m, err := GetSelfMem()
	if err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	fmt.Println(util.MemoryHuman(m.RSS))
}

func TestGetSysCPU(t *testing.T) {
	info, err := GetSysCPU(25*time.Millisecond, false)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	for _, f := range info {
		t.Log(f)
	}
}

func TestGetOsInfo(t *testing.T) {
	info, err := GetOsInfo()
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("system information is unavailable in this environment: %v", err)
		}
		t.Error(err)
		t.FailNow()
	}
	t.Log(info.Uptime, info.OS, info.Platform, info.PlatformVersion, info.KernelVersion, info.KernelArch)
}
