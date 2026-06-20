package main

import (
	"fmt"
	"strings"

	"ui_console/shared/localsearch"
)

const (
	responseLanguageZH = "zh"
	responseLanguageEN = "en"
	responseLanguagePT = "pt"
	responseLanguageJA = "ja"
	responseLanguageKO = "ko"
)

func normalizeResponseLanguage(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "en", "english", "英文":
		return responseLanguageEN
	case "pt", "pt-pt", "português", "portugues", "葡萄牙文":
		return responseLanguagePT
	case "ja", "jp", "日本語", "日文", "japanese":
		return responseLanguageJA
	case "ko", "kr", "한국어", "韓文", "韓語", "korean":
		return responseLanguageKO
	case "zh", "zh-tw", "中文", "繁中", "繁體中文", "traditional chinese", "chinese", "中":
		return responseLanguageZH
	default:
		return ""
	}
}

func isAutoLanguage(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "" || v == "auto" || v == "自動" || v == "automatic"
}

func (a *App) responseLanguage() string {
	if a != nil && a.uiSettingsService != nil {
		settings := a.uiSettingsService.Get()
		if !isAutoLanguage(settings.RoleLanguage) {
			if lang := normalizeResponseLanguage(settings.RoleLanguage); lang != "" {
				return lang
			}
		}
		if lang := normalizeResponseLanguage(settings.PanelLanguage); lang != "" {
			return lang
		}
	}
	return responseLanguageZH
}

func (a *App) routingReplyLanguageRule() string {
	switch a.responseLanguage() {
	case responseLanguageEN:
		return "回答內容請使用 English"
	case responseLanguagePT:
		return "回答內容請使用 Português"
	case responseLanguageJA:
		return "回答內容請使用日本語"
	case responseLanguageKO:
		return "回答內容請使用한국어"
	default:
		return "回答內容請使用繁體中文"
	}
}

func languageInstruction(lang string) string {
	switch normalizeResponseLanguage(lang) {
	case responseLanguageEN:
		return "English"
	case responseLanguagePT:
		return "Portuguese"
	case responseLanguageJA:
		return "Japanese"
	case responseLanguageKO:
		return "Korean"
	default:
		return "Traditional Chinese"
	}
}

func localizedOfflineChatReply(userText, lang string) (string, bool) {
	key := normalizeFastPathText(userText)
	if key == "" {
		return "", false
	}
	if len([]rune(key)) > 16 {
		return "", false
	}
	language := normalizeResponseLanguage(lang)
	switch key {
	case "你好", "妳好", "您好", "哈囉", "嗨", "hi", "hello", "hey", "yo":
		switch language {
		case responseLanguageEN:
			return "Hello! How can I help?", true
		case responseLanguagePT:
			return "Olá! Em que posso ajudar?", true
		default:
			return "你好！有什麼我可以幫你的嗎？", true
		}
	case "olá", "ola":
		if language == responseLanguagePT {
			return "Olá! Em que posso ajudar?", true
		}
	case "謝謝", "感謝", "多謝", "感恩", "thanks", "thank you", "thx", "obrigado", "obrigada":
		switch language {
		case responseLanguageEN:
			return "You're welcome!", true
		case responseLanguagePT:
			return "De nada!", true
		default:
			return "不客氣！", true
		}
	case "收到嗎", "在嗎", "還在嗎", "有在嗎", "你在嗎", "在不在":
		switch language {
		case responseLanguageEN:
			return "I'm here. Go ahead.", true
		case responseLanguagePT:
			return "Estou aqui. Diz-me.", true
		default:
			return "在的，請說。", true
		}
	case "recebeste", "recebeu":
		if language == responseLanguagePT {
			return "Recebi. Diz-me.", true
		}
	case "收到", "好的", "好", "ok", "okay", "了解", "知道了", "明白", "嗯", "沒問題":
		switch language {
		case responseLanguageEN:
			return "Okay.", true
		case responseLanguagePT:
			return "Está bem.", true
		default:
			return "好的。", true
		}
	case "掰掰", "再見", "bye", "goodbye", "晚安", "adeus", "tchau":
		switch language {
		case responseLanguageEN:
			return "Goodbye. Call me when you need me.", true
		case responseLanguagePT:
			return "Até logo. Chama-me quando precisares.", true
		default:
			return "再見，需要時再叫我。", true
		}
	}
	return "", false
}

func (a *App) offlineChatReply(userText string) (string, bool) {
	return localizedOfflineChatReply(userText, a.responseLanguage())
}

func localizedReferenceNotFound(fileName, query, lang string) string {
	switch normalizeResponseLanguage(lang) {
	case responseLanguageEN:
		return fmt.Sprintf("I couldn't find content related to %q in the reference file %q.", query, fileName)
	case responseLanguagePT:
		return fmt.Sprintf("Não encontrei conteúdo relacionado com %q no ficheiro de referência %q.", query, fileName)
	default:
		return fmt.Sprintf("在引用檔「%s」裡找不到「%s」相關內容。", fileName, query)
	}
}

func localizedNoReferenceFiles(lang string) string {
	switch normalizeResponseLanguage(lang) {
	case responseLanguageEN:
		return "I don't see any loaded reference files right now."
	case responseLanguagePT:
		return "Não vejo ficheiros de referência carregados neste momento."
	default:
		return "我這邊目前沒有看到已載入的引用檔。"
	}
}

func (a *App) formatRecentReferenceFilesAnswer(refs []routingReferenceFile) string {
	lang := a.responseLanguage()
	if len(refs) == 0 {
		return localizedNoReferenceFiles(lang)
	}
	var b strings.Builder
	switch normalizeResponseLanguage(lang) {
	case responseLanguageEN:
		b.WriteString("Loaded reference files:")
	case responseLanguagePT:
		b.WriteString("Ficheiros de referência carregados:")
	default:
		b.WriteString("有，已載入的引用檔：")
	}
	for i, ref := range refs {
		if i >= 6 {
			break
		}
		fmt.Fprintf(&b, "\n%d. %s (%s, %s)", i+1, ref.Name, strings.ToUpper(ref.Ext), ref.ModifiedAt.Format("2006-01-02 15:04:05"))
	}
	return b.String()
}

func (a *App) formatLocalSearchOutcome(req localsearch.SearchRequest, outcome localsearch.SearchOutcome) string {
	return a.formatLocalSearchOutcomeForLanguage(req, outcome, a.responseLanguage())
}

func (a *App) formatLocalSearchOutcomeForLanguage(req localsearch.SearchRequest, outcome localsearch.SearchOutcome, lang string) string {
	if normalizeResponseLanguage(lang) == responseLanguageZH {
		return localsearch.FormatSearchOutcome(req, outcome)
	}
	query := strings.TrimSpace(req.Query)
	if len(outcome.Results) == 0 {
		switch normalizeResponseLanguage(lang) {
		case responseLanguagePT:
			return fmt.Sprintf("A pesquisa local não encontrou resultados para %q.\nExperimente palavras-chave mais curtas ou indique um âmbito: memória, documentos, registos, trace ou ferramentas.", query)
		default:
			return fmt.Sprintf("Local search found no results for %q.\nTry shorter keywords or specify a scope: memory, documents, logs, trace, or tools.", query)
		}
	}
	var b strings.Builder
	switch normalizeResponseLanguage(lang) {
	case responseLanguagePT:
		fmt.Fprintf(&b, "A pesquisa local encontrou %d resultado(s) para %q:", len(outcome.Results), query)
	default:
		fmt.Fprintf(&b, "Local search found %d result(s) for %q:", len(outcome.Results), query)
	}
	for i, result := range outcome.Results {
		fmt.Fprintf(&b, "\n\n%d. ", i+1)
		if result.Source == "skill" && strings.TrimSpace(result.Title) != "" {
			if normalizeResponseLanguage(lang) == responseLanguagePT {
				b.WriteString("Skill: ")
			} else {
				b.WriteString("Skill: ")
			}
			b.WriteString(result.Title)
			if snip := strings.TrimSpace(result.Snippet); snip != "" && snip != result.Title {
				if normalizeResponseLanguage(lang) == responseLanguagePT {
					b.WriteString("\n   Resumo: ")
				} else {
					b.WriteString("\n   Summary: ")
				}
				b.WriteString(snip)
			}
			continue
		}
		if result.Snippet != "" {
			if normalizeResponseLanguage(lang) == responseLanguagePT {
				b.WriteString("Conteúdo: ")
			} else {
				b.WriteString("Content: ")
			}
			b.WriteString(result.Snippet)
		}
	}
	return b.String()
}

func (a *App) localSearchNoResultsPrompt(query string, askQuestion bool) string {
	switch a.responseLanguage() {
	case responseLanguageEN:
		if askQuestion {
			return fmt.Sprintf("I couldn't find %q in local data. Search the web instead?", query)
		}
		return fmt.Sprintf("I couldn't find %q in local data. Reply \"yes\" to search the web instead.", query)
	case responseLanguagePT:
		if askQuestion {
			return fmt.Sprintf("Não encontrei %q nos dados locais. Queres pesquisar na web?", query)
		}
		return fmt.Sprintf("Não encontrei %q nos dados locais. Responde \"sim\" para pesquisar na web.", query)
	default:
		if askQuestion {
			return fmt.Sprintf("本機資料裡找不到「%s」。要改用網路搜尋嗎？", query)
		}
		return fmt.Sprintf("本機資料沒有找到「%s」。你可以回覆「好」改用網路搜尋。", query)
	}
}
