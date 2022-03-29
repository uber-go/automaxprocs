// Copyright (c) 2022 Uber Technologies, Inc.
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
	// _cgroupv2CPUMax is the file name for the CGroup-V2 CPU max and period
	// parameter.
	_cgroupv2CPUMax = "cpu.max"
	// _cgroupFSType is the Linux CGroup-V2 file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupv2FSType = "cgroup2"

	_cgroupv2MountPoint = "/sys/fs/cgroup"

	_cgroupV2CPUMaxDefaultPeriod = 100000
	_cgroupV2CPUMaxQuotaMax      = "max"
)

const (
	_cgroupv2CPUMaxQuotaIndex = iota
	_cgroupv2CPUMaxPeriodIndex
)

// IsCGroupV2 returns true if the system supports and uses cgroup2.
// It gets the required information for deciding from mountinfo file.
func IsCGroupV2() (bool, error) {
	return isCGroupV2(_procPathMountInfo)
}

func isCGroupV2(procPathMountInfo string) (bool, error) {
	var isV2 bool
	newMountPoint := func(mp *MountPoint) error {
		if mp.FSType == _cgroupv2FSType && mp.MountPoint == _cgroupv2MountPoint {
			isV2 = true
		}
		return nil
	}
	if err := parseMountInfo(procPathMountInfo, newMountPoint); err != nil {
		return false, err
	}
	return isV2, nil
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
		if len(fields) == 0 || len(fields) > 2 {
			return -1, false, fmt.Errorf("invalid format")
		}
		if fields[_cgroupv2CPUMaxQuotaIndex] == _cgroupV2CPUMaxQuotaMax {
			return -1, false, nil
		}
		max, err := strconv.Atoi(fields[_cgroupv2CPUMaxQuotaIndex])
		if err != nil {
			return -1, false, err
		}
		var period int
		if len(fields) == 1 {
			period = _cgroupV2CPUMaxDefaultPeriod
		} else {
			period, err = strconv.Atoi(fields[_cgroupv2CPUMaxPeriodIndex])
			if err != nil {
				return -1, false, err
			}
		}
		return float64(max) / float64(period), true, nil
	}
	if err := scanner.Err(); err != nil {
		return -1, false, err
	}
	return 0, false, io.ErrUnexpectedEOF
}
