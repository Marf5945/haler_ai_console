package main

import (
	"strings"
	"testing"
	"time"

	"ui_console/adapter/visual_learning"
	"ui_console/data/conversation"
	"ui_console/shared/actionchain"
	"ui_console/shared/localsearch"
)

func chainText(action, target, next string) string {
	return action + actionchain.Separator + target + actionchain.Separator + next
}

func TestParseToolRoutingDecisionChatRequiresPrefix(t *testing.T) {
	decision := parseToolRoutingDecision("\u9592\u804a" + actionchain.Separator + "\u9019\u662f\u4e00\u822c\u56de\u7b54")
	if decision.Kind != toolRoutingDecisionChat {
		t.Fatalf("Kind = %q, want %q", decision.Kind, toolRoutingDecisionChat)
	}
	if decision.Text != "\u9019\u662f\u4e00\u822c\u56de\u7b54" {
		t.Fatalf("Text = %q", decision.Text)
	}

	plain := parseToolRoutingDecision("\u9019\u662f\u6c92\u6709\u524d\u7db4\u7684\u81ea\u7136\u8a9e\u8a00")
	if plain.Kind != toolRoutingDecisionNeedTool {
		t.Fatalf("plain natural language should fall back to need_tool, got %q", plain.Kind)
	}
}

func TestParseToolRoutingDecisionOperationAction(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u67e5\u8a62", "Chrome ChatGPT \u95dc\u9589\u5206\u9801", "\u64cd\u4f5c"))
	if decision.Kind != toolRoutingDecisionAction {
		t.Fatalf("Kind = %q, want action", decision.Kind)
	}
	if decision.Action != "\u67e5\u8a62" || decision.Target != "Chrome ChatGPT \u95dc\u9589\u5206\u9801" || decision.Next != "\u64cd\u4f5c" {
		t.Fatalf("unexpected action chain: %#v", decision)
	}
}

func TestParseSearchTermsSpaceSeparated(t *testing.T) {
	terms := parseSearchTerms("\u91cd\u73fe Chrome \u700f\u89bd\u5668 \u9ede\u64ca ChatGPT \u9801\u9762\u5167\u5bb9 \u95dc\u9589 \u5206\u9801 \u9304\u88fd \u52d5\u4f5c", "\u5e6b\u6211\u91cd\u73fe Chrome")
	if len(terms) < 8 {
		t.Fatalf("expected parsed terms, got %#v", terms)
	}
	want := map[string]bool{
		"\u91cd\u73fe":       true,
		"Chrome":             true,
		"\u700f\u89bd\u5668": true,
		"ChatGPT":            true,
		"\u95dc\u9589":       true,
		"\u9304\u88fd":       true,
		"\u52d5\u4f5c":       true,
	}
	for _, term := range terms {
		delete(want, term)
	}
	if len(want) != 0 {
		t.Fatalf("missing terms: %#v from %#v", want, terms)
	}
}

func TestInferSkillActionTargetDoesNotStealNaturalLanguage(t *testing.T) {
	app := &App{}
	if got, ok := app.inferSkillActionTarget("\u5e6b\u6211\u67e5\u8a62\u9304\u88fd\u7684\u64cd\u4f5c"); ok {
		t.Fatalf("natural language should enter three-stage routing first, got %q", got)
	}
	if got, ok := app.inferSkillActionTarget(chainText("\u67e5\u8a62", "\u5df2\u9304\u88fd\u7684\u95dc\u9589\u5206\u9801", "\u64cd\u4f5c")); !ok || got != "\u67e5\u8a62"+actionchain.Separator+"\u5df2\u9304\u88fd\u7684\u95dc\u9589\u5206\u9801" {
		t.Fatalf("explicit action-chain should still route, got %q ok=%v", got, ok)
	}
}

func TestNormalizeToolRoutingDecisionPromotesReplayQueryToOperation(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u67e5\u8a62", "Chrome ChatGPT \u95dc\u9589\u5206\u9801", "\u64cd\u4f5c"))
	lookup := toolRoutingLookupContext{
		Query: "Chrome ChatGPT \u95dc\u9589\u5206\u9801",
		Operations: []visual_learning.OperationSearchResult{
			{Tag: "demo-03677400", Title: "\u5728 Chrome \u4e2d\u64cd\u4f5c ChatGPT \u9801\u9762\u4e26\u95dc\u9589\u5206\u9801"},
		},
	}
	normalized := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u91cd\u73fe Chrome \u700f\u89bd\u5668\u4e2d\u9ede\u64ca ChatGPT \u9801\u9762\u5167\u5bb9\u5f8c\u95dc\u9589\u5206\u9801", lookup)
	if normalized.Action != "\u64cd\u4f5c" || normalized.Target != "demo-03677400" || normalized.Next != actionchain.StandbyNext {
		t.Fatalf("expected replay query to promote to operation, got %#v", normalized)
	}

	queryOnly := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u67e5\u8a62\u9304\u88fd\u7684\u64cd\u4f5c", lookup)
	if queryOnly.Action != "\u67e5\u8a62" || queryOnly.Next != "\u64cd\u4f5c" {
		t.Fatalf("query-only request should remain a query, got %#v", queryOnly)
	}
}

func TestNormalizeToolRoutingDecisionPromotesCurrentQuestionToWeb(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u641c\u5c0b", "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2", actionchain.StandbyNext))
	lookup := toolRoutingLookupContext{Query: "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2"}

	normalized := normalizeToolRoutingDecision(decision, "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2\u5982\u4f55", lookup)
	if normalized.Action != "\u7db2\u8def" || normalized.Target != "\u4eca\u5929\u7684\u661f\u5ea7\u904b\u52e2" || normalized.Next != actionchain.StandbyNext {
		t.Fatalf("expected current question to route to web search, got %#v", normalized)
	}
}

func TestNormalizeToolRoutingDecisionKeepsExplicitLocalSearchLocal(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("\u641c\u5c0b", "API key", actionchain.StandbyNext))
	lookup := toolRoutingLookupContext{Query: "API key"}

	normalized := normalizeToolRoutingDecision(decision, "\u5e6b\u6211\u672c\u6a5f\u641c\u5c0b API key", lookup)
	if normalized.Action != "\u641c\u5c0b" {
		t.Fatalf("explicit local search should remain local, got %#v", normalized)
	}
}

func TestProgramSkillRequestRepairsInsteadOfForcedNormalize(t *testing.T) {
	userText := "幫我做一個穿衣建議skill，依照天氣 JSON 和衣服表格輸出建議"
	chat := parseToolRoutingDecision("閒聊ㄌ我無法直接寫入檔案，但可以提供 Python 程式碼")
	normalized := normalizeToolRoutingDecision(chat, userText, toolRoutingLookupContext{Query: "穿衣建議"})
	if normalized.Kind != toolRoutingDecisionChat {
		t.Fatalf("normalize should not force tool route, got %#v", normalized)
	}
	if !shouldRepairToolRoutingDecision(userText, normalized) {
		t.Fatalf("program skill request should ask LLM to repair routing output")
	}
	prompt := buildToolRoutingRepairPrompt("BASE", "閒聊ㄌ我不能寫檔", userText)
	for _, want := range []string{"程式ㄌ穿衣建議ㄌ輸出", "不要產 Python", "不要嘗試 activate_skill/write_file/invoke_agent"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("repair prompt missing %q: %s", want, prompt)
		}
	}
}

func TestProgramSkillTableRequestRepairsGenericSkillOutput(t *testing.T) {
	userText := "產生 skill 輸入 電料料表"
	name, ok := inferGoProgramAuthoringRequest(userText)
	if !ok {
		t.Fatalf("table-like skill request should be recognized")
	}
	if name != "電料料表" {
		t.Fatalf("program name = %q, want 電料料表", name)
	}
	chat := parseToolRoutingDecision("閒聊ㄌ一般來說 Gemini CLI 的 Skill 會包含 skill.yaml")
	if !shouldRepairToolRoutingDecision(userText, chat) {
		t.Fatalf("generic skill.yaml answer should be repaired into app go program routing")
	}
	prompt := buildToolRoutingRepairPrompt("BASE", chat.Raw, userText)
	for _, want := range []string{"程式ㄌ電料料表ㄌ輸出", "不使用 Gemini CLI skill.yaml", "activate_skill"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("repair prompt missing %q: %s", want, prompt)
		}
	}
}

func TestExtractGoProgramNameStripsHowToWords(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"幫我做一個產出電料Bom的用法", "產出電料Bom"},
		{"做一個產出電料Bom做法", "產出電料Bom"},
		{"製作產出電料Bom作法", "產出電料Bom"},
		{"產生 skill 電料表的用途", "電料表"},
		// 不可誤傷：名稱本身不含這些詞時保持原樣。
		{"做一個產出電料Bom", "產出電料Bom"},
	}
	for _, c := range cases {
		if got := extractGoProgramName(c.in); got != c.want {
			t.Fatalf("extractGoProgramName(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestGoProgramAuthoringClarifiesMissingFormat(t *testing.T) {
	question, need := goProgramAuthoringClarification("產生 skill 輸入 電料料表")
	if !need {
		t.Fatalf("missing input/output format should ask for clarification")
	}
	for _, want := range []string{"輸入格式", "資料範例", "輸出的欄位"} {
		if !strings.Contains(question, want) {
			t.Fatalf("clarification missing %q: %s", want, question)
		}
	}
	_, need = goProgramAuthoringClarification("幫我做一個穿衣建議程式，依照天氣 JSON 和衣服表格輸出建議")
	if need {
		t.Fatalf("explicit weather JSON plus clothing table request should enter authoring loop")
	}
}

func TestRoutingLookupContextIncludesLoadedFilesCompactly(t *testing.T) {
	stoppedAt := time.Date(2026, 6, 7, 0, 21, 34, 0, time.FixedZone("CST", 8*60*60))
	lookup := toolRoutingLookupContext{
		Query: "看到 拉進來的檔案",
		Terms: []string{"看到", "拉進來的檔案"},
		RecentReferences: []routingReferenceFile{{
			Name:       "電料BOM-260327M1.xlsx",
			Ext:        "xlsx",
			ModifiedAt: time.Date(2026, 6, 9, 22, 48, 3, 0, time.FixedZone("CST", 8*60*60)),
		}},
		RecentOperations: []visual_learning.LearningRunCatalogItem{{
			Tag:       "demo-123",
			RunID:     "learn-should-not-leak",
			Title:     "empty demo",
			Summary:   "No replayable click steps were recorded.",
			Keywords:  []string{"empty", "no", "replayable"},
			StepCount: 0,
			StoppedAt: &stoppedAt,
		}},
		LocalMatches: []localsearch.SearchResult{{
			Source: "document",
			Title:  "最近引用文件: 電料BOM-260327M1.xlsx",
			Path:   "/tmp/電料BOM-260327M1.xlsx",
			Score:  34,
		}},
	}
	out := formatToolRoutingLookupContext(lookup)
	for _, want := range []string{"loaded_files=", "電料BOM-260327M1.xlsx", "2026-06-09T22:48:03+08:00"} {
		if !strings.Contains(out, want) {
			t.Fatalf("routing lookup missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"learn-should-not-leak", "keywords=", "summary="} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("routing lookup should stay compact, found %q:\n%s", unwanted, out)
		}
	}
}

func TestRoutingPromptsSkipPersonaAndCurrentHistoryDuplicate(t *testing.T) {
	userText := "你有看到我拉進來的檔案嗎"
	recent := []conversation.Sentence{
		{Role: "user", Content: "上一輪說明"},
		{Role: "user", Content: userText},
	}
	keywordPrompt := buildSearchTermExtractionPrompt("身份=成熟可靠的本機助手", userText, recent)
	if strings.Contains(keywordPrompt, "身份=成熟可靠") {
		t.Fatalf("keyword routing prompt should not include persona: %s", keywordPrompt)
	}
	if strings.Count(keywordPrompt, userText) != 1 {
		t.Fatalf("keyword routing prompt should include current input once, got %d:\n%s", strings.Count(keywordPrompt, userText), keywordPrompt)
	}
	judgePrompt := buildToolRoutingDecisionPrompt("角色=憂樂傻酷", userText, "[lookup] loaded_files=name=\"demo.xlsx\" ext=\"xlsx\" mtime=\"2026-06-09T22:48:03+08:00\"\n[/lookup]", recent)
	if strings.Contains(judgePrompt, "角色=憂樂傻酷") {
		t.Fatalf("judge routing prompt should not include persona: %s", judgePrompt)
	}
	if strings.Count(judgePrompt, userText) != 1 {
		t.Fatalf("judge routing prompt should include current input once, got %d:\n%s", strings.Count(judgePrompt, userText), judgePrompt)
	}
}

func TestNormalizeReferenceVisibilityQuestionUsesLoadedFiles(t *testing.T) {
	decision := parseToolRoutingDecision(chainText("提問", "請問檔案名稱或路徑是什麼？", actionchain.StandbyNext))
	lookup := toolRoutingLookupContext{
		Query: "看到 拉進來的檔案",
		RecentReferences: []routingReferenceFile{{
			Name:       "demo.xlsx",
			Ext:        "xlsx",
			ModifiedAt: time.Now(),
		}},
	}
	normalized := normalizeToolRoutingDecision(decision, "你有看到我拉進來的檔案嗎", lookup)
	if normalized.Action != "搜尋" || normalized.Target != "引用文件" || normalized.Next != "文件" {
		t.Fatalf("loaded-file visibility question should route to references, got %#v", normalized)
	}
}

func TestGoProgramContractReviewPromptCatchesClothingTableSimplification(t *testing.T) {
	manifest := seedGoProgramManifest("program-test", "穿衣建議", t.TempDir())
	prompt := buildGoProgramContractReviewPrompt(
		"幫我做一個穿衣建議skill，依照天氣 JSON 和衣服表格輸出建議",
		manifest,
		"func main(){ /* only reads temperature */ }",
		[]byte(`{"result":"天氣溫和，建議穿薄外套"}`),
	)
	for _, want := range []string{"天氣 JSON + 衣服表格", "只用 temperature", "ok=false"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("contract review prompt missing %q: %s", want, prompt)
		}
	}
	review, err := parseGoProgramContractReview(`{"ok":false,"reason":"缺衣服表格","feedback":"請使用 clothing_items","missing_user_data":true,"required_data":["衣服表格"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !requiresClothingTable(review) {
		t.Fatalf("expected clothing-table review to request clothing CSV template")
	}
}
