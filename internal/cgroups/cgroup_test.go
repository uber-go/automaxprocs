// +build linux

package cgroups

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCGroupParamPath(t *testing.T) {
	cgroup := NewCGroup("/sys/fs/cgroup/cpu")
	assert.Equal(t, "/sys/fs/cgroup/cpu", cgroup.Path())
	assert.Equal(t, "/sys/fs/cgroup/cpu/cpu.cfs_quota_us", cgroup.ParamPath("cpu.cfs_quota_us"))
}

func TestCGroupReadFirstLine(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedContent string
		shouldHaveError bool
	}{
		{
			name:            "cpu",
			paramName:       "cpu.cfs_period_us",
			expectedContent: "100000",
			shouldHaveError: false,
		},
		{
			name:            "absent",
			paramName:       "cpu.stat",
			expectedContent: "",
			shouldHaveError: true,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedContent: "",
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroup := NewCGroup(cgroupPath)

		content, err := cgroup.readFirstLine(tt.paramName)
		assert.Equal(t, tt.expectedContent, content, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupReadInt(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedValue   int
		shouldHaveError bool
	}{
		{
			name:            "cpu",
			paramName:       "cpu.cfs_period_us",
			expectedValue:   100000,
			shouldHaveError: false,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "invalid",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "absent",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroup := NewCGroup(cgroupPath)

		value, err := cgroup.readInt(tt.paramName)
		assert.Equal(t, tt.expectedValue, value, "%s/%s", tt.name, tt.paramName)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
