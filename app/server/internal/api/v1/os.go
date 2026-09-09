package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/pkg/ecode"
	"github.com/iyear/pure-live-core/pkg/format"
	"github.com/iyear/pure-live-core/service/svc_os"
)

func GetOSInfo(c *gin.Context) {
	info, err := svc_os.GetOSInfo()
	if err != nil {
		format.HTTP(c, ecode.ErrorGetOSInfo, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, info)
}

func GetSysMem(c *gin.Context) {
	r, err := svc_os.GetSysMem()
	if err != nil {
		format.HTTP(c, ecode.ErrorGetSysMem, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, r)
}

func GetSelfMem(c *gin.Context) {
	r, err := svc_os.GetSelfMem()
	if err != nil {
		format.HTTP(c, ecode.ErrorGetSelfMem, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, r)
}

func GetSysCPU(c *gin.Context) {
	r, err := svc_os.GetSysCPU()
	if err != nil {
		format.HTTP(c, ecode.ErrorGetSysCPU, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, r)
}

func GetSelfCPU(c *gin.Context) {
	r, err := svc_os.GetSelfCPU()
	if err != nil {
		format.HTTP(c, ecode.ErrorGetSelfCPU, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, r)
}

// GetOSAll preserves supported metrics when a platform cannot provide another
// metric (for example, system CPU usage in a CGO-disabled macOS binary).
func GetOSAll(c *gin.Context) {
	metrics := gin.H{}
	unavailable := []string{}
	collectOSMetric(metrics, &unavailable, "info", svc_os.GetOSInfo)
	collectOSMetric(metrics, &unavailable, "sys_cpu", svc_os.GetSysCPU)
	collectOSMetric(metrics, &unavailable, "self_cpu", svc_os.GetSelfCPU)
	collectOSMetric(metrics, &unavailable, "sys_mem", svc_os.GetSysMem)
	collectOSMetric(metrics, &unavailable, "self_mem", svc_os.GetSelfMem)
	if len(metrics) == 0 {
		format.HTTP(c, ecode.ErrorGetOsAll, fmt.Errorf("system metrics are unavailable"), nil)
		return
	}
	if len(unavailable) > 0 {
		metrics["unavailable"] = unavailable
	}
	metrics["lan_ip"] = svc_os.LANIPv4()
	format.HTTP(c, ecode.Success, nil, metrics)
}

func collectOSMetric[T any](metrics gin.H, unavailable *[]string, name string, read func() (T, error)) {
	value, err := read()
	if err != nil {
		*unavailable = append(*unavailable, name)
		return
	}
	metrics[name] = value
}
