// +build !linux

package runtime

// CPUQuotaToGOMAXPROCS converts the CPU quota applied to the calling process
// to a valid GOMAXPROCS value. This is Linux-specific and not supported in the
// current OS.
func CPUQuotaToGOMAXPROCS(_ int) (int, CPUQuotaStatus, error) {
	return -1, CPUQuotaUndefined, nil
}
