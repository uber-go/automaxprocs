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
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{
			name:    "mountinfo",
			want:    false,
			wantErr: false,
		},
		{
			name:    "mountinfo-v1-v2",
			want:    false,
			wantErr: false,
		},
		{
			name:    "mountinfo-v2",
			want:    true,
			wantErr: false,
		},
		{
			name:    "mountinfo-nonexistent",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountInfoPath := filepath.Join(testDataProcPath, "v2", tt.name)
			isV2, err := isCGroupV2(mountInfoPath)

			assert.Equal(t, tt.want, isV2, tt.name)

			if tt.wantErr {
				assert.Error(t, err, tt.name)
			} else {
				assert.NoError(t, err, tt.name)
			}
		})
	}
}

func TestCGroupsCPUQuotaV2(t *testing.T) {
	tests := []struct {
		name    string
		want    float64
		wantOK  bool
		wantErr bool
	}{
		{
			name:    "set",
			want:    2.5,
			wantOK:  true,
			wantErr: false,
		},
		{
			name:    "unset",
			want:    -1.0,
			wantOK:  false,
			wantErr: false,
		},
		{
			name:    "only-max",
			want:    5.0,
			wantOK:  true,
			wantErr: false,
		},
		{
			name:    "invalid-max",
			want:    -1.0,
			wantOK:  false,
			wantErr: true,
		},
		{
			name:    "invalid-period",
			want:    -1.0,
			wantOK:  false,
			wantErr: true,
		},
		{
			name:    "nonexistent",
			want:    -1.0,
			wantOK:  false,
			wantErr: false,
		},
	}

	cgroupPath := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quota, defined, err := cpuQuotaV2(cgroupPath, tt.name)
			assert.Equal(t, tt.want, quota, tt.name)
			assert.Equal(t, tt.wantOK, defined, tt.name)

			if tt.wantErr {
				assert.Error(t, err, tt.name)
			} else {
				assert.NoError(t, err, tt.name)
			}
		})
	}
}
