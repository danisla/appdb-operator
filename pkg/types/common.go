package types

// ProvisioningStatus represents the string mapping to the possible status.Provisioning values. See the const definition below for enumerated states.
type ProvisioningStatus string

const (
	ProvisioningStatusPending  = "PENDING"
	ProvisioningStatusFailed   = "FAILED"
	ProvisioningStatusComplete = "COMPLETE"
)
