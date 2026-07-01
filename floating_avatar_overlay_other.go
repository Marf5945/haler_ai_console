//go:build !windows

package main

func showFloatingAvatarOverlay(imagePath string, x int, y int, maxW int, maxH int, onAction func(string)) error {
	return nil
}

func closeFloatingAvatarOverlay() {}

func floatingAvatarOverlayPosition() (int, int) {
	return 0, 0
}
