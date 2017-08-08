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

package automaxprocs

import (
	"os"
	"runtime"

	iruntime "go.uber.org/automaxprocs/internal/runtime"
)

const _maxProcsKey = "GOMAXPROCS"

func currentMaxProcs() int {
	return runtime.GOMAXPROCS(0)
}

type config struct {
	printf func(string, ...interface{})
	procs  func(int) (int, iruntime.CPUQuotaStatus, error)
}

func (c *config) log(fmt string, args ...interface{}) {
	if c.printf != nil {
		c.printf(fmt, args...)
	}
}

// An Option alters the behavior of Set.
type Option interface {
	apply(*config)
}

// Logger uses the supplied printf implementation for log output. By default,
// Set doesn't log anything.
func Logger(printf func(string, ...interface{})) Option {
	return optionFunc(func(cfg *config) {
		cfg.printf = printf
	})
}

type optionFunc func(*config)

func (of optionFunc) apply(cfg *config) { of(cfg) }

// Set GOMAXPROCS to match the Linux container CPU quota (if any), returning
// any error encountered and an undo function.
//
// Set is a no-op on non-Linux systems and in Linux environments without a
// configured CPU quota.
func Set(opts ...Option) (func(), error) {
	cfg := &config{procs: iruntime.CPUQuotaToGOMAXPROCS}
	for _, o := range opts {
		o.apply(cfg)
	}

	prev := currentMaxProcs()
	undo := func() {
		cfg.log("resetting GOMAXPROCS to %d", prev)
		runtime.GOMAXPROCS(prev)
	}

	// Honor the GOMAXPROCS environment variable if present. Otherwise, amend
	// `runtime.GOMAXPROCS()` with the current process' CPU quota if the OS is
	// Linux, and guarantee a minimum value of 2 to ensure efficiency.
	if max, exists := os.LookupEnv(_maxProcsKey); exists {
		cfg.log("GOMAXPROCS=%d: honoring explicitly-configured GOMAXPROCS from environment", max)
		return undo, nil
	}

	maxProcs, status, err := cfg.procs(iruntime.MinGOMAXPROCS)
	switch {
	case err != nil:
		return undo, err
	case status == iruntime.CPUQuotaUndefined:
		cfg.log("GOMAXPROCS=%d: CPU quota undefined", currentMaxProcs())
	case status == iruntime.CPUQuotaMinUsed:
		cfg.log("GOMAXPROCS=%d: using minimum allowed GOMAXPROCS", currentMaxProcs())
	default:
		cfg.log("GOMAXPROCS=%d: determined from CPU quota", currentMaxProcs())
	}

	runtime.GOMAXPROCS(maxProcs)
	return undo, nil
}
