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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCGroupsIsCGroupV2(t *testing.T) {
	testTable := []struct {
		name            string
		expectedIsV2    bool
		shouldHaveError bool
	}{
		{
			name:            "mountinfo",
			expectedIsV2:    false,
			shouldHaveError: false,
		},
		{
			name:            "mountinfo-v1-v2",
			expectedIsV2:    false,
			shouldHaveError: false,
		},
		{
			name:            "mountinfo-v2",
			expectedIsV2:    true,
			shouldHaveError: false,
		},
		{
			name:            "mountinfo-nonexistent",
			expectedIsV2:    false,
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		mountInfoPath := filepath.Join(testDataProcPath, "v2", tt.name)
		isV2, err := isCGroupV2(mountInfoPath)

		assert.Equal(t, tt.expectedIsV2, isV2, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupsCPUQuotaV2(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   float64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "set",
			expectedQuota:   2.5,
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "unset",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: false,
		},
		{
			name:            "only-max",
			expectedQuota:   5.0,
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "invalid-max",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: true,
		},
		{
			name:            "invalid-period",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: true,
		},
	}

	quota, defined, err := cpuQuotaV2("nonexistent", "nonexistent")
	assert.Equal(t, -1.0, quota, "nonexistent")
	assert.Equal(t, false, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	cgroupPath := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range testTable {
		quota, defined, err := cpuQuotaV2(cgroupPath, tt.name)
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
