//go:build windows

package voice

func availableDiskBytes(path string) (uint64, bool) {
	return 0, false
}
