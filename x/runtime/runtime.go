package runtime

// CPUQuotaStatus presents the status of how CPU quota is used
type CPUQuotaStatus int

const (
	// CPUQuotaUndefined is returned when CPU quota is undefined
	CPUQuotaUndefined CPUQuotaStatus = iota
	// CPUQuotaUsed is returned when a valid CPU quota can be used
	CPUQuotaUsed
	// CPUQuotaMinUsed is return when CPU quota is larger than the min value
	CPUQuotaMinUsed
)

// MinGOMAXPROCS defines the minimum value for GOMAXPROCS
const MinGOMAXPROCS = 2
