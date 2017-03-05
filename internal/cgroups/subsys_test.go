// +build linux

package cgroups

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCGroupSubsysFromLine(t *testing.T) {
	testTable := []struct {
		name           string
		line           string
		expectedSubsys *CGroupSubsys
	}{
		{
			name: "single-subsys",
			line: "1:cpu:/",
			expectedSubsys: &CGroupSubsys{
				ID:         1,
				Subsystems: []string{"cpu"},
				Name:       "/",
			},
		},
		{
			name: "multi-subsys",
			line: "8:cpu,cpuacct,cpuset:/docker/1234567890abcdef",
			expectedSubsys: &CGroupSubsys{
				ID:         8,
				Subsystems: []string{"cpu", "cpuacct", "cpuset"},
				Name:       "/docker/1234567890abcdef",
			},
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Equal(t, tt.expectedSubsys, subsys, tt.name)
		assert.NoError(t, err, tt.name)
	}
}

func TestNewCGroupSubsysFromLineErr(t *testing.T) {
	lines := []string{
		"1:cpu",
		"1:cpu,cpuacct:/:/necessary-field",
		"not-a-number:cpu:/",
	}
	_, parseError := strconv.Atoi("not-a-number")

	testTable := []struct {
		name          string
		line          string
		expectedError error
	}{
		{
			name:          "fewer-fields",
			line:          lines[0],
			expectedError: cgroupSubsysFormatInvalidError{lines[0]},
		},
		{
			name:          "more-fields",
			line:          lines[1],
			expectedError: cgroupSubsysFormatInvalidError{lines[1]},
		},
		{
			name:          "illegal-id",
			line:          lines[2],
			expectedError: parseError,
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Nil(t, subsys, tt.name)
		assert.Equal(t, tt.expectedError, err, tt.name)
	}
}
