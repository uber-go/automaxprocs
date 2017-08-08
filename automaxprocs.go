// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// +build linux

package automaxprocs

import (
	"log"
	"os"
	"runtime"

	iruntime "go.uber.org/automaxprocs/internal/runtime"
)

func init() {
	// Honor the GOMAXPROCS environment variable if present. Otherwise, amend
	// `runtime.GOMAXPROCS()` with the current process' CPU quota if the OS is
	// Linux, and guarantee a minimum value of 2 to ensure efficiency.
	if _, exists := os.LookupEnv("GOMAXPROCS"); exists {
		return
	}

	maxProcs, status, err := iruntime.CPUQuotaToGOMAXPROCS(iruntime.MinGOMAXPROCS)
	switch {
	case err != nil:
		log.Printf("GOMAXPROCS=%d: Error on reading CPU quota: %v", runtime.GOMAXPROCS(0), err)
	case status == iruntime.CPUQuotaUndefined:
		log.Printf("GOMAXPROCS=%d: CPU quota undefined", runtime.GOMAXPROCS(0))
	case status == iruntime.CPUQuotaMinUsed:
		runtime.GOMAXPROCS(maxProcs)
		log.Printf("GOMAXPROCS=%d: Min value for GOMAXPROCS chosen over lower CPU quota in favor of parallelism", runtime.GOMAXPROCS(0))
	default:
		runtime.GOMAXPROCS(maxProcs)
		log.Printf("GOMAXPROCS=%d: Determined from CPU quota", runtime.GOMAXPROCS(0))
	}
}
