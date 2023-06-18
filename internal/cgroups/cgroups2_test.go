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
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCGroupsIsCGroupV2(t *testing.T) {
	tests := []struct {
		name    string
		isV2    bool
		wantErr bool // should be false if isV2 is true
	}{
		{
			name:    "mountinfo",
			isV2:    false,
			wantErr: false,
		},
		{
			name:    "mountinfo-v1-v2",
			isV2:    false,
			wantErr: false,
		},
		{
			name:    "mountinfo-v2",
			isV2:    true,
			wantErr: false,
		},
		{
			name:    "mountinfo-nonexistent",
			isV2:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountInfoPath := filepath.Join(testDataProcPath, "v2", tt.name)
			procCgroupPath := filepath.Join(testDataProcPath, "v2", "cgroup-root")
			_, err := newCGroups2From(mountInfoPath, procCgroupPath)
			switch {
			case tt.wantErr:
				assert.Error(t, err)
			case !tt.isV2:
				assert.ErrorIs(t, err, ErrNotV2)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestCGroupsCPUQuotaV2(t *testing.T) {
	tests := []struct {
		name    string
		want    float64
		wantOK  bool
		wantErr string
	}{
		{
			name:   "set",
			want:   2.5,
			wantOK: true,
		},
		{
			name:   "unset",
			want:   -1.0,
			wantOK: false,
		},
		{
			name:   "only-max",
			want:   5.0,
			wantOK: true,
		},
		{
			name:    "invalid-max",
			wantErr: `parsing "asdf": invalid syntax`,
		},
		{
			name:    "invalid-period",
			wantErr: `parsing "njn": invalid syntax`,
		},
		{
			name:   "nonexistent",
			want:   -1.0,
			wantOK: false,
		},
		{
			name:    "empty",
			wantErr: "unexpected EOF",
		},
		{
			name:    "too-few-fields",
			wantErr: "invalid format",
		},
		{
			name:    "too-many-fields",
			wantErr: "invalid format",
		},
		{
			name:    "zero-period",
			wantErr: "zero value for period is not allowed",
		},
	}

	mountPoint := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quota, defined, err := (&CGroups2{
				mountPoint: mountPoint,
				groupPath:  "/",
				cpuMaxFile: tt.name,
			}).CPUQuota()

			if len(tt.wantErr) > 0 {
				require.Error(t, err, tt.name)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.want, quota, tt.name)
				assert.Equal(t, tt.wantOK, defined, tt.name)
			}
		})
	}
}

func TestCGroup2GroupPathDiscovery(t *testing.T) {
	tests := []struct {
		procCgroup string
		wantPath   string
	}{
		{
			procCgroup: "cgroup-root",
			wantPath:   "/",
		},
		{
			procCgroup: "cgroup-subdir",
			wantPath:   "/Example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.procCgroup, func(t *testing.T) {
			mountInfoPath := filepath.Join(testDataProcPath, "v2", "mountinfo-v2")
			procCgroupPath := filepath.Join(testDataProcPath, "v2", tt.procCgroup)
			cgroups, err := newCGroups2From(mountInfoPath, procCgroupPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, cgroups.groupPath)
		})
	}
}

func TestCGroup2GroupPathDiscovery_Errors(t *testing.T) {
	t.Run("no matching subsystem", func(t *testing.T) {
		mountInfoPath := filepath.Join(testDataProcPath, "v2", "mountinfo-v2")
		procCgroupPath := filepath.Join(testDataProcPath, "v2", "cgroup-no-match")
		_, err := newCGroups2From(mountInfoPath, procCgroupPath)
		assert.ErrorIs(t, err, ErrNotV2)
	})

	t.Run("invalid subsystems", func(t *testing.T) {
		mountInfoPath := filepath.Join(testDataProcPath, "v2", "mountinfo-v2")
		procCgroupPath := filepath.Join(testDataProcPath, "v2", "cgroup-invalid")
		_, err := newCGroups2From(mountInfoPath, procCgroupPath)
		assert.Contains(t, err.Error(), "invalid format for CGroupSubsys")
	})
}

func TestCGroupsCPUQuotaV2_OtherErrors(t *testing.T) {
	t.Run("no permissions to open", func(t *testing.T) {
		if u, err := user.Current(); err == nil && u.Uid == "0" {
			t.Skip("running as root, test skipped")
		}

		t.Parallel()

		const name = "foo"

		mountPoint := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(mountPoint, name), nil /* write only*/, 0222))

		_, _, err := (&CGroups2{mountPoint: mountPoint, groupPath: "/", cpuMaxFile: name}).CPUQuota()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})
}
