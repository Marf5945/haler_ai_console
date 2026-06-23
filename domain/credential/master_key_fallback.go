//go:build !darwin && !windows && !linux

package credential

func NewOSMasterKeyProvider(_ string, _ string, _ string) MasterKeyProvider {
	return unavailableMasterKeyProvider{id: "unsupported_os"}
}
