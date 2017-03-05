// +build linux

package cgroups

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMountPointFromLine(t *testing.T) {
	testTable := []struct {
		name     string
		line     string
		expected *MountPoint
	}{
		{
			name: "root",
			line: "1 0 252:0 / / rw,noatime - ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
			expected: &MountPoint{
				MountID:        1,
				ParentID:       0,
				DeviceID:       "252:0",
				Root:           "/",
				MountPoint:     "/",
				Options:        []string{"rw", "noatime"},
				OptionalFields: []string{},
				FSType:         "ext4",
				MountSource:    "/dev/dm-0",
				SuperOptions:   []string{"rw", "errors=remount-ro", "data=ordered"},
			},
		},
		{
			name: "cgroup",
			line: "31 23 0:24 /docker /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime shared:1 - cgroup cgroup rw,cpu",
			expected: &MountPoint{
				MountID:        31,
				ParentID:       23,
				DeviceID:       "0:24",
				Root:           "/docker",
				MountPoint:     "/sys/fs/cgroup/cpu",
				Options:        []string{"rw", "nosuid", "nodev", "noexec", "relatime"},
				OptionalFields: []string{"shared:1"},
				FSType:         "cgroup",
				MountSource:    "cgroup",
				SuperOptions:   []string{"rw", "cpu"},
			},
		},
	}

	for _, tt := range testTable {
		mountPoint, err := NewMountPointFromLine(tt.line)
		assert.Equal(t, tt.expected, mountPoint, tt.name)
		assert.NoError(t, err, tt.name)
	}
}

func TestNewMountPointFromLineErr(t *testing.T) {
	linesWithInvalidIDs := []string{
		"invalidMountID 0 252:0 / / rw,noatime - ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
		"1 invalidParentID 252:0 / / rw,noatime - ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
		"invalidMountID invalidParentID 252:0 / / rw,noatime - ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
	}

	for i, line := range linesWithInvalidIDs {
		mountPoint, err := NewMountPointFromLine(line)
		assert.Nil(t, mountPoint, "[%d] %q", i, line)
		assert.Error(t, err, line)
	}

	linesWithInvalidFields := []string{
		"1 0 252:0 / / rw,noatime ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
		"1 0 252:0 / / rw,noatime shared:1 - ext4 /dev/dm-0",
		"1 0 252:0 / / rw,noatime shared:1 ext4 - /dev/dm-0 rw,errors=remount-ro,data=ordered",
		"1 0 252:0 / / rw,noatime shared:1 ext4 /dev/dm-0 rw,errors=remount-ro,data=ordered",
		"random line",
	}

	for i, line := range linesWithInvalidFields {
		mountPoint, err := NewMountPointFromLine(line)
		errExpected := mountPointFormatInvalidError{line}

		assert.Nil(t, mountPoint, "[%d] %q", i, line)
		assert.Equal(t, err, errExpected, "[%d] %q", i, line)
	}
}

func TestMountPointTranslate(t *testing.T) {
	line := "31 23 0:24 /docker/0123456789abcdef /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime shared:1 - cgroup cgroup rw,cpu"
	cgroupMountPoint, err := NewMountPointFromLine(line)

	assert.NotNil(t, cgroupMountPoint)
	assert.NoError(t, err)

	testTable := []struct {
		name            string
		pathToTranslate string
		pathTranslated  string
	}{
		{
			name:            "root",
			pathToTranslate: "/docker/0123456789abcdef",
			pathTranslated:  "/sys/fs/cgroup/cpu",
		},
		{
			name:            "root-with-extra-slash",
			pathToTranslate: "/docker/0123456789abcdef/",
			pathTranslated:  "/sys/fs/cgroup/cpu",
		},
		{
			name:            "descendant-from-root",
			pathToTranslate: "/docker/0123456789abcdef/large/cpu.cfs_quota_us",
			pathTranslated:  "/sys/fs/cgroup/cpu/large/cpu.cfs_quota_us",
		},
	}

	for _, tt := range testTable {
		path, err := cgroupMountPoint.Translate(tt.pathToTranslate)
		assert.Equal(t, tt.pathTranslated, path, tt.name)
		assert.NoError(t, err, tt.name)
	}
}

func TestMountPointTranslateError(t *testing.T) {
	line := "31 23 0:24 /docker/0123456789abcdef /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime shared:1 - cgroup cgroup rw,cpu"
	cgroupMountPoint, err := NewMountPointFromLine(line)

	assert.NotNil(t, cgroupMountPoint)
	assert.NoError(t, err)

	inaccessiblePaths := []string{
		"/",
		"/docker",
		"/docker/0123456789abcdef-let-me-hack-this-path",
		"/docker/0123456789abcde/abc/../../def",
		"/system.slice/docker.service",
	}

	for i, path := range inaccessiblePaths {
		translated, err := cgroupMountPoint.Translate(path)
		errExpected := pathNotExposedFromMountPointError{
			mountPoint: cgroupMountPoint.MountPoint,
			root:       cgroupMountPoint.Root,
			path:       path,
		}

		assert.Equal(t, "", translated, "inaccessiblePaths[%d] == %q", i, path)
		assert.Equal(t, errExpected, err, "inaccessiblePaths[%d] == %q", i, path)
	}

	relPaths := []string{
		"docker",
		"docker/0123456789abcde/large",
		"system.slice/docker.service",
	}

	for i, path := range relPaths {
		translated, err := cgroupMountPoint.Translate(path)

		assert.Equal(t, "", translated, "relPaths[%d] == %q", i, path)
		assert.Error(t, err, path)
	}
}
