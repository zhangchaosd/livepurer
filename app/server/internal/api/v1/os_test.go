package v1

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOSMetricFailurePreservesSupportedValues(t *testing.T) {
	metrics := gin.H{}
	unavailable := []string{}
	collectOSMetric(metrics, &unavailable, "info", func() (string, error) { return "darwin", nil })
	collectOSMetric(metrics, &unavailable, "sys_cpu", func() (float64, error) { return 0, errors.New("not implemented") })
	collectOSMetric(metrics, &unavailable, "sys_mem", func() (int, error) { return 100, nil })
	if !reflect.DeepEqual(metrics, gin.H{"info": "darwin", "sys_mem": 100}) {
		t.Fatalf("unsupported metrics must not appear as zero values: %#v", metrics)
	}
	if !reflect.DeepEqual(unavailable, []string{"sys_cpu"}) {
		t.Fatalf("missing unavailable metric: %#v", unavailable)
	}
}
