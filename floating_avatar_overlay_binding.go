package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ui_console/adapter/persona_avatar"
)

func (a *App) EnterFloatingAvatarOverlay(personaID string, mode string, state string, x int, y int) error {
	imagePath, maxW, maxH, err := a.resolveFloatingAvatarOverlayImage(personaID, mode, state)
	if err != nil {
		return err
	}
	overlayMode := normalizeOverlayMode(mode)
	return showFloatingAvatarOverlay(imagePath, overlayMode, x, y, maxW, maxH, a.handleFloatingAvatarOverlayAction)
}

func (a *App) EnterFloatingAvatarOverlayImage(imageData []byte, mode string, x int, y int) error {
	if len(imageData) == 0 {
		return fmt.Errorf("floating avatar overlay image is empty")
	}
	cacheDir := filepath.Join(appDataRoot(), "data", "cache", "floating_avatar_overlay")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	overlayMode := normalizeOverlayMode(mode)
	out := filepath.Join(cacheDir, fmt.Sprintf("overlay_%s.png", overlayMode))
	if err := os.WriteFile(out, imageData, 0o600); err != nil {
		return err
	}
	maxW, maxH := overlayModeSize(overlayMode)
	return showFloatingAvatarOverlay(out, overlayMode, x, y, maxW, maxH, a.handleFloatingAvatarOverlayAction)
}

func (a *App) ExitFloatingAvatarOverlay() error {
	closeFloatingAvatarOverlay()
	return nil
}

func (a *App) GetFloatingAvatarOverlayPosition() map[string]int {
	x, y := floatingAvatarOverlayPosition()
	return map[string]int{"x": x, "y": y}
}

func (a *App) SetFloatingAvatarOverlayChatMode(enabled bool) {
	setFloatingAvatarOverlayChatMode(enabled)
}

func (a *App) SetFloatingAvatarOverlayMetadata(personaName string, replyText string, placeholder string) {
	setFloatingAvatarOverlayMetadata(personaName, replyText, placeholder)
}

func (a *App) handleFloatingAvatarOverlayAction(action string, text string) {
	if action == "" {
		action = "restore"
	}
	a.eventBus.Emit("floating_avatar:menu_action", map[string]string{"action": action, "text": text})
}

func (a *App) resolveFloatingAvatarOverlayImage(personaID string, mode string, state string) (string, int, int, error) {
	personaID = strings.TrimSpace(personaID)
	if personaID == "" {
		personaID = "persona-a"
	}
	state = normalizeFloatingAvatarState(state)
	config := a.avatarService.GetCurrentAvatar(personaID)
	pack := persona_avatar.NormalizePixelPack(personaID, config.PixelPack)
	if pack == "" {
		pack = persona_avatar.DefaultPixelPack(personaID)
	}

	overlayMode := normalizeOverlayMode(mode)
	if overlayMode == "full" {
		if path := findOverlayAssetPath("persona_fullbody", overlayPackFolder(pack), overlayFullBodyName(state)); path != "" {
			maxW, maxH := overlayModeSize(overlayMode)
			return path, maxW, maxH, nil
		}
	}

	if config.AvatarProvider == persona_avatar.ProviderStaticImage && config.StaticAvatarPath != "" {
		if path := resolveOverlayDataPath(config.StaticAvatarPath); path != "" {
			return path, 96, 96, nil
		}
	}

	if path := findOverlayAssetPath("persona_avatars", overlayPackFolder(pack), state+".png"); path != "" {
		return path, 96, 96, nil
	}
	if path := findOverlayAssetPath("persona_avatars", overlayPackFolder(pack), "idle.png"); path != "" {
		return path, 96, 96, nil
	}

	data, err := a.renderPixelAvatarPack(pack, state, 96)
	if err != nil {
		return "", 0, 0, err
	}
	cacheDir := filepath.Join(appDataRoot(), "data", "cache", "floating_avatar_overlay")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", 0, 0, err
	}
	out := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_%s.png", personaID, pack, state))
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return "", 0, 0, err
	}
	return out, 96, 96, nil
}

func normalizeOverlayMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "full":
		return "full"
	}
	return "head"
}

func overlayModeSize(mode string) (int, int) {
	switch normalizeOverlayMode(mode) {
	case "full":
		return 200, 360
	}
	return 96, 96
}

func normalizeFloatingAvatarState(state string) string {
	switch strings.TrimSpace(state) {
	case "thinking", "working", "happy", "warning", "blocked", "sleepy", "sad", "speechless":
		return state
	default:
		return "idle"
	}
}

func overlayPackFolder(pack string) string {
	switch pack {
	case "uncle":
		return "uncle_bust"
	case "wolf":
		return "wolfdog"
	default:
		return pack
	}
}

func overlayFullBodyName(state string) string {
	if state == "blocked" || state == "warning" {
		return "fullbody.png"
	}
	return "fullbody_idle.png"
}

func resolveOverlayDataPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return existingFile(path)
	}
	return existingFile(filepath.Join(appDataRoot(), filepath.FromSlash(path)))
}

func findOverlayAssetPath(parts ...string) string {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates, filepath.Join(append([]string{wd, "frontend", "src", "assets"}, parts...)...))
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(append([]string{exeDir, "frontend", "src", "assets"}, parts...)...),
			filepath.Join(append([]string{filepath.Dir(exeDir), "frontend", "src", "assets"}, parts...)...),
		)
	}
	for _, candidate := range candidates {
		if path := existingFile(candidate); path != "" {
			return path
		}
	}
	return ""
}

func existingFile(path string) string {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return ""
}
