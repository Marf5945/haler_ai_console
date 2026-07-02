//go:build !windows

package main

func showFloatingAvatarOverlay(imagePath string, mode string, x int, y int, maxW int, maxH int, onAction func(string, string)) error {
	return nil
}

func closeFloatingAvatarOverlay() {}

func floatingAvatarOverlayPosition() (int, int) {
	return 0, 0
}

func setFloatingAvatarOverlayChatMode(enabled bool) {}

func setFloatingAvatarOverlayMetadata(personaName string, replyText string, placeholder string) {}
