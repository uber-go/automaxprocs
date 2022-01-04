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

//go:build linux
// +build linux

package cgroups

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	// _cgroupFSType is the Linux CGroup file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupFSType = "cgroup"
	// _cgroupSubsysCPU is the CPU CGroup subsystem.
	_cgroupSubsysCPU = "cpu"
	// _cgroupSubsysCPUAcct is the CPU accounting CGroup subsystem.
	_cgroupSubsysCPUAcct = "cpuacct"
	// _cgroupSubsysCPUSet is the CPUSet CGroup subsystem.
	_cgroupSubsysCPUSet = "cpuset"
	// _cgroupSubsysMemory is the Memory CGroup subsystem.
	_cgroupSubsysMemory = "memory"

	// _cgroupCPUCFSQuotaUsParam is the file name for the CGroup CFS quota
	// parameter.
	_cgroupCPUCFSQuotaUsParam = "cpu.cfs_quota_us"
	// _cgroupCPUCFSPeriodUsParam is the file name for the CGroup CFS period
	// parameter.
	_cgroupCPUCFSPeriodUsParam = "cpu.cfs_period_us"

	// _cgroupv2CPUMax is the file name for the CGroup-V2 CPU max and period
	// parameter.
	_cgroupv2CPUMax = "cpu.max"
	// _cgroupFSType is the Linux CGroup-V2 file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupv2FSType = "cgroup2"
)

const (
	_procPathCGroup     = "/proc/self/cgroup"
	_procPathMountInfo  = "/proc/self/mountinfo"
	_cgroupv2MountPoint = "/sys/fs/cgroup"
)

const (
	_cgroupv2CPUMaxQuota = iota
	_cgroupv2CPUMaxPeriod
)

// CGroups is a map that associates each CGroup with its subsystem name.
type CGroups map[string]*CGroup

// NewCGroups returns a new *CGroups from given `mountinfo` and `cgroup` files
// under for some process under `/proc` file system (see also proc(5) for more
// information).
func NewCGroups(procPathMountInfo, procPathCGroup string) (CGroups, error) {
	cgroupSubsystems, err := parseCGroupSubsystems(procPathCGroup)
	if err != nil {
		return nil, err
	}

	cgroups := make(CGroups)
	newMountPoint := func(mp *MountPoint) error {
		if mp.FSType != _cgroupFSType {
			return nil
		}

		for _, opt := range mp.SuperOptions {
			subsys, exists := cgroupSubsystems[opt]
			if !exists {
				continue
			}

			cgroupPath, err := mp.Translate(subsys.Name)
			if err != nil {
				return err
			}
			cgroups[opt] = NewCGroup(cgroupPath)
		}

		return nil
	}

	if err := parseMountInfo(procPathMountInfo, newMountPoint); err != nil {
		return nil, err
	}
	return cgroups, nil
}

// NewCGroupsForCurrentProcess returns a new *CGroups instance for the current
// process.
func NewCGroupsForCurrentProcess() (CGroups, error) {
	return NewCGroups(_procPathMountInfo, _procPathCGroup)
}

// CPUQuota returns the CPU quota applied with the CPU cgroup controller.
// It is a result of `cpu.cfs_quota_us / cpu.cfs_period_us`. If the value of
// `cpu.cfs_quota_us` was not set (-1), the method returns `(-1, nil)`.
func (cg CGroups) CPUQuota() (float64, bool, error) {
	cpuCGroup, exists := cg[_cgroupSubsysCPU]
	if !exists {
		return -1, false, nil
	}

	cfsQuotaUs, err := cpuCGroup.readInt(_cgroupCPUCFSQuotaUsParam)
	if defined := cfsQuotaUs > 0; err != nil || !defined {
		return -1, defined, err
	}

	cfsPeriodUs, err := cpuCGroup.readInt(_cgroupCPUCFSPeriodUsParam)
	if err != nil {
		return -1, false, err
	}

	return float64(cfsQuotaUs) / float64(cfsPeriodUs), true, nil
}

// IsCGroupV2 returns true if the system supports and uses cgroup2.
// It gets the required information for deciding from mountinfo file.
func IsCGroupV2() (bool, error) {
	return isCGroupV2(_procPathMountInfo)
}

func isCGroupV2(procPathMountInfo string) (bool, error) {
	isV2 := false
	newMountPoint := func(mp *MountPoint) error {
		if mp.FSType == _cgroupv2FSType && mp.MountPoint == _cgroupv2MountPoint {
			isV2 = true
		}
		return nil
	}
	if err := parseMountInfo(procPathMountInfo, newMountPoint); err != nil {
		return false, err
	}
	if isV2 {
		return true, nil
	}
	return false, nil
}

// CPUQuotaV2 returns the CPU quota applied with the CPU cgroup2 controller.
// It is a result of reading cpu quota and period from cpu.max file.
// It will return `cpu.max / cpu.period`. If cpu.max is set to max, it returns
// (-1, false, nil)
func CPUQuotaV2() (float64, bool, error) {
	return cpuQuotaV2(_cgroupv2MountPoint, _cgroupv2CPUMax)
}

func cpuQuotaV2(cgroupv2MountPoint, cgroupv2CPUMax string) (float64, bool, error) {
	cpuMaxParams, err := os.Open(path.Join(cgroupv2MountPoint, cgroupv2CPUMax))
	if err != nil {
		if os.IsNotExist(err) {
			return -1, false, nil
		}
		return -1, false, err
	}
	scanner := bufio.NewScanner(cpuMaxParams)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			return -1, false, fmt.Errorf("invalid format")
		}
		if fields[0] == "max" {
			return -1, false, nil
		}
		max, err := strconv.Atoi(fields[_cgroupv2CPUMaxQuota])
		if err != nil {
			return -1, false, err
		}
		period, err := strconv.Atoi(fields[_cgroupv2CPUMaxPeriod])
		if err != nil {
			return -1, false, err
		}
		return float64(max) / float64(period), true, nil
	}
	if err := scanner.Err(); err != nil {
		return -1, false, err
	}
	return 0, false, io.ErrUnexpectedEOF
}
