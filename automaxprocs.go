// +build linux

package automaxprocs

import (
	"log"
	"os"
	"runtime"

	xruntime "github.com/uber-go/automaxprocs/x/runtime"
)

func init() {
	// Honor the GOMAXPROCS environment variable if present. Otherwise, amend
	// `runtime.GOMAXPROCS()` with the current process' CPU quota if the OS is
	// Linux, and guarantee a minimum value of 2 to ensure efficiency.
	if _, exists := os.LookupEnv("GOMAXPROCS"); exists {
		return
	}

	maxProcs, status, err := xruntime.CPUQuotaToGOMAXPROCS(xruntime.MinGOMAXPROCS)
	switch {
	case err != nil:
		log.Printf("GOMAXPROCS=%d: Error on reading CPU quota: %v", runtime.GOMAXPROCS(0), err)
	case status == xruntime.CPUQuotaUndefined:
		log.Printf("GOMAXPROCS=%d: CPU quota undefined", runtime.GOMAXPROCS(0))
	case status == xruntime.CPUQuotaMinUsed:
		runtime.GOMAXPROCS(maxProcs)
		log.Printf("GOMAXPROCS=%d: Min value for GOMAXPROCS chosen over lower CPU quota in favor of parallelism", runtime.GOMAXPROCS(0))
	default:
		runtime.GOMAXPROCS(maxProcs)
		log.Printf("GOMAXPROCS=%d: Determined from CPU quota", runtime.GOMAXPROCS(0))
	}
}
