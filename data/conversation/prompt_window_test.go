// prompt_window_test.go — 滑動視窗單元測試(token 優化熱點 1)。
package conversation

import (
	"fmt"
	"strings"
	"testing"
)

func mkSent(id, role, content string) Sentence {
	return Sentence{ID: id, Role: role, Content: content}
}

// 產生 n 輪 Q/A,每句 content 固定長度。
func mkTurns(n, runesPerSentence int) []Sentence {
	var out []Sentence
	for i := 1; i <= n; i++ {
		body := strings.Repeat("字", runesPerSentence)
		out = append(out, mkSent(fmt.Sprintf("[I-%03d]", i), "user", fmt.Sprintf("問%d:%s", i, body)))
		out = append(out, mkSent(fmt.Sprintf("[O-%03d]", i), "assistant", fmt.Sprintf("答%d:%s", i, body)))
	}
	return out
}

func TestPromptWindowDisabledReturnsAll(t *testing.T) {
	old := PromptWindowEnabled
	PromptWindowEnabled = false
	defer func() { PromptWindowEnabled = old }()

	raw := mkTurns(20, 500)
	res := ApplyPromptWindow(raw, DefaultPromptWindowConfig())
	if len(res.Sentences) != len(raw) || res.Elided != 0 {
		t.Fatalf("disabled window should return all %d sentences, got %d (elided=%d)", len(raw), len(res.Sentences), res.Elided)
	}
}

func TestPromptWindowShortHistoryUnchanged(t *testing.T) {
	raw := mkTurns(3, 100) // 3 輪 < KeepRecentTurns(4),應全保留
	res := ApplyPromptWindow(raw, DefaultPromptWindowConfig())
	if len(res.Sentences) != len(raw) || res.Elided != 0 {
		t.Fatalf("short history should be unchanged, got %d/%d (elided=%d)", len(res.Sentences), len(raw), res.Elided)
	}
}

func TestPromptWindowKeepsRecentTurnsRegardlessOfBudget(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	cfg.RawBudgetRunes = 0 // 預算歸零,只剩無條件保留區
	raw := mkTurns(10, 200)
	res := ApplyPromptWindow(raw, cfg)

	// 最近 4 輪 = 8 句必須完整存在
	for i := len(raw) - 8; i < len(raw); i++ {
		found := false
		for _, s := range res.Sentences {
			if s.ID == raw[i].ID && s.Content == raw[i].Content {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("recent sentence %s should be kept intact", raw[i].ID)
		}
	}
	if res.Elided == 0 {
		t.Fatalf("older sentences should be elided with zero budget")
	}
}

func TestPromptWindowBudgetEvictsOldestFirst(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	cfg.RawBudgetRunes = 500 // 約可容納 2 句 200-rune 的較早句
	raw := mkTurns(10, 200)
	res := ApplyPromptWindow(raw, cfg)

	if res.Elided == 0 {
		t.Fatalf("expected eviction with tight budget")
	}
	// 被剔除的應是最舊的句子:第 1 輪一定不在,視窗內最舊的保留句必比被剔除者新
	for _, s := range res.Sentences {
		if s.ID == "[I-001]" || s.ID == "[O-001]" {
			t.Fatalf("oldest sentences should be evicted first, found %s", s.ID)
		}
	}
}

func TestPromptWindowProtectsImportantSentences(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	cfg.RawBudgetRunes = 0 // 預算歸零仍須保留豁免句
	raw := mkTurns(10, 200)
	// 在最舊處插入三種豁免句
	protected := []Sentence{
		mkSent("[I-000]", "user", "[系統提供: operation_candidates] 重要系統上下文"),
		mkSent("[tool-action: 002]", "tool-action", "工具動作紀錄"),
		mkSent("[I-000b]", "user", "task-intent-執行排程任務"),
	}
	raw = append(protected, raw...)
	res := ApplyPromptWindow(raw, cfg)

	for _, p := range protected {
		found := false
		for _, s := range res.Sentences {
			if s.ID == p.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("protected sentence %s should survive zero budget", p.ID)
		}
	}
}

func TestPromptWindowTruncatesLongOldSentence(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	raw := mkTurns(6, 50)
	// 第 1 輪的 assistant 回答塞成 2000 rune(在最近 4 輪之外)
	raw[1].Content = strings.Repeat("長", 2000)
	res := ApplyPromptWindow(raw, cfg)

	for _, s := range res.Sentences {
		if s.ID == "[O-001]" {
			n := len([]rune(s.Content))
			limit := cfg.HeadRunes + cfg.TailRunes + len([]rune("……(中略)……"))
			if n > limit {
				t.Fatalf("long old sentence should be truncated to <=%d runes, got %d", limit, n)
			}
			if !strings.Contains(s.Content, "(中略)") {
				t.Fatalf("truncated sentence should contain elision marker")
			}
			return
		}
	}
	t.Fatalf("[O-001] should still be in window (budget allows)")
}

func TestPromptWindowInsertsPlaceholderAndKeepsOrder(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	cfg.RawBudgetRunes = 100
	raw := mkTurns(10, 200)
	res := ApplyPromptWindow(raw, cfg)

	if res.Elided == 0 {
		t.Fatalf("expected elision")
	}
	first := res.Sentences[0]
	if first.ID != "[window]" || !strings.Contains(first.Content, fmt.Sprintf("%d 句", res.Elided)) {
		t.Fatalf("first sentence should be placeholder with elided count, got %+v", first)
	}
	// 其餘句子須維持原相對順序
	lastIdx := -1
	for _, s := range res.Sentences[1:] {
		for i, r := range raw {
			if r.ID == s.ID {
				if i < lastIdx {
					t.Fatalf("sentence order broken at %s", s.ID)
				}
				lastIdx = i
				break
			}
		}
	}
}

func TestPromptWindowRecentSentencesNotTruncated(t *testing.T) {
	cfg := DefaultPromptWindowConfig()
	raw := mkTurns(6, 50)
	// 最後一輪 assistant 回答超長:在最近 4 輪內,不得截斷
	raw[len(raw)-1].Content = strings.Repeat("近", 2000)
	res := ApplyPromptWindow(raw, cfg)

	last := res.Sentences[len(res.Sentences)-1]
	if len([]rune(last.Content)) != 2000 {
		t.Fatalf("recent long sentence must stay intact, got %d runes", len([]rune(last.Content)))
	}
}
