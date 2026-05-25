// persona_avatar/delete.go — 刪除靜態頭像（§10.8）。
// 刪除後自動 fallback 到 pixel avatar。
package persona_avatar

import (
	"fmt"
	"os"
	"time"
)

// DeleteStaticAvatar 刪除指定 persona 的靜態頭像圖片。
// 刪除後 provider 自動切換為 built_in_pixel。
func (s *Service) DeleteStaticAvatar(personaID string) error {
	if personaID == lockedAvatarPersonaID {
		return fmt.Errorf("persona avatar is locked: %s", personaID)
	}

	dir := s.avatarDir(personaID)

	// 刪除靜態圖片檔案（忽略不存在的錯誤）
	filesToRemove := []string{
		"avatar_static.png",
		"avatar_static_original.png",
		"avatar_preview_cache.png",
	}
	for _, f := range filesToRemove {
		path := dir + "/" + f
		os.Remove(path) // 忽略錯誤
	}

	// 更新 config → fallback 到 pixel
	config := s.GetCurrentAvatar(personaID)
	config.AvatarProvider = ProviderPixel
	config.StaticAvatarPath = ""
	config.OriginalImagePath = ""
	config.Crop = nil
	config.UpdatedAt = time.Now().Format(time.RFC3339)

	return s.saveConfig(personaID, config)
}
