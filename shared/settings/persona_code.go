package settings

// persona_code.go — 人格「編號」。
//
// 每個人格有一組亂數 6 碼（shared/shortcode，去除容易看混的 0/O/1/I）。
// 編號在兩種情況下會重新產生：
//   1. 新增/複製進來一個人格（SavePersona 的「新增」分支）。
//   2. 偵測到「重灌本程式，或整份資料夾被複製到別台機器／別的使用者設定檔」
//      ——本機安裝識別碼（shared/installepoch）跟這份資料裡記錄的
//      InstallEpoch 對不上時，全部人格的編號會一次重新產生。
// 一般的關閉/重開程式、或編輯既有人格的資料，都不會動到既有編號。
//
// 每次產生新編號都會檢查跟目前所有人格的編號不重複。

import (
	"ui_console/shared/installepoch"
	"ui_console/shared/shortcode"
)

func existingPersonaCodes(personas []Persona) map[string]bool {
	existing := make(map[string]bool, len(personas))
	for _, p := range personas {
		if p.Code != "" {
			existing[p.Code] = true
		}
	}
	return existing
}

// assignNewPersonaCode 給「新增/複製進來」的人格產生一個保證不重複的編號。
func assignNewPersonaCode(personas []Persona) string {
	return shortcode.Generate(existingPersonaCodes(personas))
}

// ensurePersonaCodes 確保每個人格都有編號；若偵測到重灌或複製到新機器，
// 會把全部人格的編號重新產生一輪。回傳是否有變動。
func ensurePersonaCodes(state *State) bool {
	machineID, err := installepoch.LocalMachineID()
	if err != nil || machineID == "" {
		// 拿不到機器識別碼就不強制判斷重灌/複製，至少補齊缺號的人格。
		return fillMissingPersonaCodes(state)
	}

	if state.InstallEpoch == machineID {
		return fillMissingPersonaCodes(state)
	}

	// InstallEpoch 對不上：這份資料第一次在本機出現，或是重灌/複製過來的。
	existing := make(map[string]bool, len(state.Personas))
	for i := range state.Personas {
		code := shortcode.Generate(existing)
		state.Personas[i].Code = code
		existing[code] = true
	}
	state.InstallEpoch = machineID
	return true
}

func fillMissingPersonaCodes(state *State) bool {
	changed := false
	existing := existingPersonaCodes(state.Personas)
	for i := range state.Personas {
		if state.Personas[i].Code == "" {
			code := shortcode.Generate(existing)
			state.Personas[i].Code = code
			existing[code] = true
			changed = true
		}
	}
	return changed
}
