// builtin_manifests.go — 內建能力 manifest 定義與註冊。
// 啟動時由 app.go 呼叫 RegisterDocumentBuiltins，不寫磁碟。
package skill_step

// RegisterDocumentBuiltins 註冊全部內建 manifest。
// TASK 31 / Phase 0.4：每個 builtin 缺 lifecycle，註冊前補安全預設——
// 低風險唯讀的 auto_execute=true；workspace_write/medium（write/export/scheduler）為 false。
func RegisterDocumentBuiltins(r *Router) {
	for _, m := range []*SkillManifest{
		builtinDocImport(), builtinDocRead(), builtinDocWrite(), builtinDocExport(),
		builtinLocalSearch(), builtinWebSearch(), builtinScheduler(), builtinGitStatus(),
		builtinTranslation(),
	} {
		EnsureLifecycle(m) // nil → DefaultLifecycle(依風險/權限決定 auto_execute)
		r.RegisterBuiltin(m)
	}
}

// builtinTranslation 是純文字轉換能力，不需要網路、檔案或程式執行權限。
// 多語 action tags 讓使用者切換介面語言後仍能穩定路由到同一個 skill。
func builtinTranslation() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.translation",
		DisplayName:   "翻譯",
		Description:   "將使用者提供的文字忠實翻譯成指定語言，保留格式、專有名詞與插值參數。",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"transform"},
			ActionTag: []string{
				"翻譯", "翻成", "譯成", "translate", "translation",
				"翻訳", "번역", "traducir", "traduzir", "แปล", "ترجمة",
			},
			DomainTag: []string{
				"language", "translation", "multilingual", "text", "localization",
				"語言", "文字", "多語", "在地化",
			},
			RiskTag: []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "none", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns: []string{
				"翻譯ㄌ文字", "翻成ㄌ英文", "translateㄌtext", "translationㄌtext",
				"翻訳ㄌテキスト", "번역ㄌ텍스트", "traducirㄌtexto", "traduzirㄌtexto",
				"แปลㄌข้อความ", "ترجمةㄌالنص",
			},
			TargetAliases: []string{
				"文字", "內容", "語言", "text", "content", "language",
				"英文", "中文", "日文", "韓文", "西班牙文", "葡萄牙文", "泰文", "阿拉伯文",
			},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinDocImport() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.document.import",
		DisplayName:   "文件匯入",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"import"},
			ActionTag:  []string{"匯入", "import", "拖入", "開啟"},
			DomainTag:  []string{"document", "文件", "檔案"},
			RiskTag:    []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_read", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"匯入文件", "import document", "開啟檔案"},
			TargetAliases:    []string{"文件", "document", "file", "檔案"},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinDocRead() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.document.read",
		DisplayName:   "文件讀取",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"lookup"},
			ActionTag:  []string{"讀取", "read", "查看", "列出"},
			DomainTag:  []string{"document", "文件"},
			RiskTag:    []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_read", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"讀取文件", "read document", "列出文件"},
			TargetAliases:    []string{"文件", "document", "content", "內容"},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinDocWrite() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.document.write",
		DisplayName:   "文件寫入",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"transform"},
			ActionTag:  []string{"寫入", "write", "儲存", "save", "建立"},
			DomainTag:  []string{"document", "文件"},
			RiskTag:    []string{"medium"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_write", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"寫入文件", "save document", "建立文件"},
			TargetAliases:    []string{"文件", "document", "file"},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinDocExport() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.document.export",
		DisplayName:   "文件匯出",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"export"},
			ActionTag:  []string{"匯出", "export", "下載", "download"},
			DomainTag:  []string{"document", "文件"},
			RiskTag:    []string{"medium"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_write", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"匯出文件", "export document", "下載文件"},
			TargetAliases:    []string{"文件", "document", "file"},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinLocalSearch() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.local.search",
		DisplayName:   "本機搜尋",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"lookup"},
			ActionTag:  []string{"本機搜尋", "搜尋", "查找", "查詢", "search", "find", "query"},
			DomainTag:  []string{"local", "本機", "記憶", "文件", "紀錄", "trace", "工具"},
			RiskTag:    []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_read", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"本機搜尋ㄌ記憶", "搜尋ㄌ文件", "查找ㄌtrace", "查詢ㄌAPI key", "searchㄌnotes"},
			TargetAliases:    []string{"記憶", "文件", "紀錄", "trace", "工具", "skill", "memory", "document"},
			MinimumAutoScore: 0.8,
		},
	}
}

func builtinWebSearch() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.web.search",
		DisplayName:   "網路搜尋",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"lookup", "research"},
			ActionTag:  []string{"網路", "網路搜尋", "搜尋網路", "查網路", "上網查", "web_search", "search_web"},
			DomainTag:  []string{"web", "internet", "latest", "current", "news", "網路", "即時資料"},
			RiskTag:    []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "web", Filesystem: "none", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"網路ㄌ最新資料", "網路搜尋ㄌ最新資料", "搜尋網路ㄌ即時資料", "web_searchㄌlatest docs", "search_webㄌcurrent information"},
			TargetAliases:    []string{"web", "internet", "latest", "current", "news", "網路", "最新", "即時"},
			MinimumAutoScore: 0.8,
		},
	}
}

// builtinScheduler 註冊排程相關 action tags，讓 LLM 知道有定時能力。
func builtinScheduler() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.scheduler",
		DisplayName:   "時間排程",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"automation"},
			ActionTag:  []string{"排程", "定時", "計時", "提醒", "schedule"},
			DomainTag:  []string{"scheduler", "排程", "時間", "提醒", "定時"},
			RiskTag:    []string{"medium"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_write", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"排程ㄌ每天", "定時ㄌ提醒", "計時ㄌ倒數", "提醒ㄌ明天"},
			TargetAliases:    []string{"排程", "定時", "提醒", "時間", "每天", "每小時", "schedule"},
			MinimumAutoScore: 0.8,
		},
	}
}

// builtinGitStatus 註冊版控狀態查詢能力，讓 LLM 可用 git status --short。
func builtinGitStatus() *SkillManifest {
	return &SkillManifest{
		SchemaVersion: "skill_manifest.v1",
		SkillID:       "builtin.git.status",
		DisplayName:   "版控狀態",
		Version:       "1.0.0",
		Tags: SkillTags{
			PurposeTag: []string{"lookup"},
			ActionTag:  []string{"版控", "git"},
			DomainTag:  []string{"git", "版控", "版本", "修改", "變更"},
			RiskTag:    []string{"low"},
		},
		Permissions: SkillPermissions{
			Network: "none", Filesystem: "workspace_read", Execution: "none",
		},
		Routing: SkillRouting{
			ActionPatterns:   []string{"版控ㄌ狀態", "版控ㄌ查看", "gitㄌstatus"},
			TargetAliases:    []string{"狀態", "修改", "變更", "status", "改了什麼"},
			MinimumAutoScore: 0.8,
		},
	}
}
