// +build linux

package cgroups

import (
	"os"
	"path/filepath"
)

var (
	pwd                 = mustGetWd()
	testDataPath        = filepath.Join(pwd, "testdata")
	testDataCGroupsPath = filepath.Join(testDataPath, "cgroups")
	testDataProcPath    = filepath.Join(testDataPath, "proc")
)

func mustGetWd() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return pwd
}
