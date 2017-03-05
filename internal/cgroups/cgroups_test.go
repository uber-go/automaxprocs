// +build linux

package cgroups

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCGroups(t *testing.T) {
	cgroupsProcCGroupPath := filepath.Join(testDataProcPath, "cgroups", "cgroup")
	cgroupsProcMountInfoPath := filepath.Join(testDataProcPath, "cgroups", "mountinfo")

	testTable := []struct {
		subsys string
		path   string
	}{
		{_cgroupSubsysCPU, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUAcct, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUSet, "/sys/fs/cgroup/cpuset"},
		{_cgroupSubsysMemory, "/sys/fs/cgroup/memory/large"},
	}

	cgroups, err := NewCGroups(cgroupsProcMountInfoPath, cgroupsProcCGroupPath)
	assert.Equal(t, len(testTable), len(cgroups))
	assert.NoError(t, err)

	for _, tt := range testTable {
		cgroup, exists := cgroups[tt.subsys]
		assert.Equal(t, true, exists, "%q expected to present in `cgroups`", tt.subsys)
		assert.Equal(t, tt.path, cgroup.path, "%q expected for `cgroups[%q].path`, got %q", tt.path, tt.subsys, cgroup.path)
	}
}

func TestNewCGroupsWithErrors(t *testing.T) {
	testTable := []struct {
		mountInfoPath string
		cgroupPath    string
	}{
		{"non-existing-file", "/dev/null"},
		{"/dev/null", "non-existing-file"},
		{
			"/dev/null",
			filepath.Join(testDataProcPath, "invalid-cgroup", "cgroup"),
		},
		{
			filepath.Join(testDataProcPath, "invalid-mountinfo", "mountinfo"),
			"/dev/null",
		},
		{
			filepath.Join(testDataProcPath, "untranslatable", "mountinfo"),
			filepath.Join(testDataProcPath, "untranslatable", "cgroup"),
		},
	}

	for _, tt := range testTable {
		cgroups, err := NewCGroups(tt.mountInfoPath, tt.cgroupPath)
		assert.Nil(t, cgroups)
		assert.Error(t, err)
	}
}

func TestCGroupsCPUQuota(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   float64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "cpu",
			expectedQuota:   6.0,
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "undefined",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: false,
		},
		{
			name:            "undefined-period",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: true,
		},
	}

	cgroups := make(CGroups)

	quota, defined, err := cgroups.CPUQuota()
	assert.Equal(t, -1.0, quota, "nonexistent")
	assert.Equal(t, false, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroups[_cgroupSubsysCPU] = NewCGroup(cgroupPath)

		quota, defined, err := cgroups.CPUQuota()
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
