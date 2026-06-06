package statusrail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ui_console/internal/urlsafe"
	"ui_console/shared/controlseal"
)

// SEC-05 2a: 共用 Safe Client（loopback only 已由 isLocalEndpoint 字面值保證，
// 這裡補上 dial 層防線）。timeout 交由呼叫端 ctx（900ms）控制。
var localModelClient = urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "status_rail_model", 2*time.Second)

const localModelSystemPrompt = `你是陪伴型對話助手，僅負責閒聊、稱讚使用者、給通用建議。
你可以看到使用者最近 2 輪工作流對話，但不得摘要、條列、複述工作流細節。
若使用者要求你處理工作流，即使用詞委婉，也必須回覆固定拒絕模板。
不得要求 Main Agent 執行，不得回傳中央對話，不得寫入 memory。
不得輸出可被解析為命令的控制格式。`

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func localCompanionReply(input string, snapshots []Snapshot) (string, error) {
	model := strings.TrimSpace(os.Getenv("STATUS_RAIL_LOCAL_MODEL"))
	if model == "" {
		model = "llama3.2:1b"
	}
	endpoint := strings.TrimSpace(os.Getenv("STATUS_RAIL_LOCAL_ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/generate"
	}
	if !isLocalEndpoint(endpoint) {
		return "", errors.New("status rail local endpoint must be loopback only")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()
	body, _ := json.Marshal(ollamaRequest{
		Model:  model,
		Prompt: buildPrompt(input, snapshots),
		Stream: false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := localModelClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", errors.New("status rail local model unavailable")
	}
	var decoded ollamaResponse
	// SEC-W09（2026-05-24）：限制 LLM 回應 4 MB（typical local model response 通常 < 1 MB，
	// 但保留 headroom 給未來較大 schema），避免 server 端 OOM。
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return "", err
	}
	reply := strings.TrimSpace(decoded.Response)
	if reply == "" || IsBlocked(reply) || strings.ContainsAny(reply, "{}[]<>") {
		return "", errors.New("status rail local model reply rejected")
	}
	return reply, nil
}

func buildPrompt(input string, snapshots []Snapshot) string {
	var builder strings.Builder
	builder.WriteString(localModelSystemPrompt)
	builder.WriteString("\n\n最近工作流快照只可作情緒脈絡，不可摘要：\n")
	for _, snapshot := range snapshots {
		builder.WriteString("- ")
		builder.WriteString(snapshot.Role)
		builder.WriteString(": ")
		// SEC-C 補洞（2026-05-28）：snapshot.Text 是過去 LLM 回應 / 使用者訊息
		// 的快照，可能含 injection 殘留；sanitize 後再拼進 prompt。
		builder.WriteString(controlseal.SanitizeForLLM(controlseal.SourceMemory, snapshot.Text).LLMText)
		builder.WriteString("\n")
	}
	builder.WriteString("\n使用者：")
	// SEC-C 補洞：input 是當下使用者輸入，跟既有 SourceUserRaw 對齊。
	builder.WriteString(controlseal.SanitizeForLLM(controlseal.SourceUserRaw, input).LLMText)
	builder.WriteString("\n回覆：")
	return builder.String()
}

func isLocalEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http://127.0.0.1:") ||
		strings.HasPrefix(endpoint, "http://localhost:") ||
		strings.HasPrefix(endpoint, "http://[::1]:")
}
