package main

import "testing"

func TestFloatingCandidatesFromQuestionTarget(t *testing.T) {
	question, candidates := floatingCandidatesFromQuestionTarget("你想查哪裡的天氣？#查目前位置=查詢天氣 目前位置#輸入地點=input:我想查天氣，地點是 #取消=不用了#多餘=不該出現")
	if question != "你想查哪裡的天氣？" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	if candidates[0].Label != "查目前位置" || candidates[0].Draft != "查詢天氣 目前位置" {
		t.Fatalf("first candidate = %+v", candidates[0])
	}
	if candidates[1].Draft != "input:我想查天氣，地點是" {
		t.Fatalf("second candidate draft = %q", candidates[1].Draft)
	}
}

func TestFloatingCandidatesFromOptionTarget(t *testing.T) {
	question, candidates := floatingCandidatesFromOptionTarget("ㄤ台北ㄤ本地ㄤ台中")
	if question != "請選擇：" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	if candidates[0].Label != "台北" || candidates[0].Draft != "台北" {
		t.Fatalf("first candidate = %+v", candidates[0])
	}
}

func TestFloatingCandidatesFromOptionTargetWithQuestion(t *testing.T) {
	question, candidates := floatingCandidatesFromOptionTarget("請選城市ㄤ台北ㄤ台中")
	if question != "請選城市" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 2 || candidates[1].Label != "台中" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestFloatingCandidatesFromOptionTargetKeepsFirstPlainOption(t *testing.T) {
	question, candidates := floatingCandidatesFromOptionTarget("紅色ㄤ綠色ㄤ藍色")
	if question != "請選擇：" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	if candidates[0].Label != "紅色" || candidates[1].Label != "綠色" || candidates[2].Label != "藍色" {
		t.Fatalf("plain options should all remain candidates: %+v", candidates)
	}
}

func TestFloatingCandidatesStripInternalControlSeal(t *testing.T) {
	question, candidates := floatingCandidatesFromOptionTarget("ㄈㄒㄜ請選城市ㄤㄈㄒㄜ台北ㄤ台中")
	if question != "請選城市" {
		t.Fatalf("question = %q", question)
	}
	if len(candidates) != 2 || candidates[0].Draft != "台北" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestQuestionCandidatesStripInternalControlSealFromDraft(t *testing.T) {
	_, candidates := floatingCandidatesFromQuestionTarget("請選#星座=ㄈㄒㄜ 查詢 今日星座運勢#輸入=input:ㄈㄒㄜ 查詢 ")
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	if candidates[0].Draft != "查詢 今日星座運勢" {
		t.Fatalf("first draft = %q", candidates[0].Draft)
	}
	if candidates[1].Draft != "input:查詢" {
		t.Fatalf("second draft = %q", candidates[1].Draft)
	}
}
